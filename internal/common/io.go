package common

type Printlnfer interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

var Silent Printlnfer = silent{}

type silent struct{}

func (silent) Print(v ...interface{}) {}

func (silent) Printf(format string, v ...interface{}) {}

func (silent) Println(v ...interface{}) {}
