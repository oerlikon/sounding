package main

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"sounding/internal/exchange"
)

func Books(listeners []exchange.Listener) []<-chan *exchange.BookUpdate {
	books := make([]<-chan *exchange.BookUpdate, 0, len(listeners))
	for _, listener := range listeners {
		if listener == nil {
			continue
		}
		if bc := listener.Book(); bc != nil {
			books = append(books, bc)
		}
	}
	if len(books) == 0 {
		return nil
	}
	return books
}

func BooksLoop(books []<-chan *exchange.BookUpdate, w io.StringWriter, wg *sync.WaitGroup) {
	cases := make([]reflect.SelectCase, len(books))
	for i, bc := range books {
		cases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(bc),
		}
	}
	var b strings.Builder
	for len(cases) > 0 {
		n, value, ok := reflect.Select(cases)
		if !ok {
			cases = append(cases[:n], cases[n+1:]...)
			continue
		}
		b.Reset()
		bu := value.Interface().(*exchange.BookUpdate)
		for _, pl := range bu.Bids {
			fmt.Fprintf(&b, "B %d,%s,%s,%s,%s,%s,%s\n",
				bu.Timestamp.UnixMilli(),
				bu.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
				bu.Exchange,
				strings.ToUpper(bu.Symbol),
				"BID",
				pl.Price,
				pl.Quantity)
		}
		for _, pl := range bu.Asks {
			fmt.Fprintf(&b, "B %d,%s,%s,%s,%s,%s,%s\n",
				bu.Timestamp.UnixMilli(),
				bu.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
				bu.Exchange,
				strings.ToUpper(bu.Symbol),
				"ASK",
				pl.Price,
				pl.Quantity)
		}
		w.WriteString(b.String())
	}
	wg.Done()
}
