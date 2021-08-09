package timestamp

import "time"

type Timestamp int64

func Stamp(t time.Time) Timestamp {
	if t.IsZero() {
		return 0
	}
	return Timestamp(t.UnixNano())
}

func Milli(ms int64) Timestamp {
	return Timestamp(ms * 1_000_000)
}

func (t Timestamp) Time() time.Time {
	return time.Unix(0, int64(t)).UTC()
}

func (t Timestamp) Unix() int {
	return int(t / 1_000_000_000)
}

func (t Timestamp) UnixMilli() int64 {
	return int64(t / 1_000_000)
}

func (t Timestamp) Add(d time.Duration) Timestamp {
	return Timestamp(int64(t) + int64(d))
}

func (t Timestamp) Sub(u Timestamp) time.Duration {
	return time.Duration(t - u)
}

func (t Timestamp) Format(layout string) string {
	return t.Time().Format(layout)
}

func (t Timestamp) S() string {
	if t == 0 {
		return "0"
	}
	return t.Format("2006-01-02_15:04:05")
}
