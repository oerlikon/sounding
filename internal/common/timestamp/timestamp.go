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
	return Timestamp(ms * 1e3)
}

func Micro(us int64) Timestamp {
	return Timestamp(us * 1e6)
}

func Float(s float64) Timestamp {
	return Timestamp(s * 1e9)
}

func FloatMilli(ms float64) Timestamp {
	return Timestamp(ms * 1e3)
}

func FloatMicro(us float64) Timestamp {
	return Timestamp(us * 1e6)
}

func (t Timestamp) Time() time.Time {
	return time.Unix(0, int64(t)).UTC()
}

func (t Timestamp) Unix() int {
	return int(t / 1e9)
}

func (t Timestamp) UnixMilli() int64 {
	return int64(t / 1e3)
}

func (t Timestamp) UnixMicro() int64 {
	return int64(t / 1e6)
}

func (t Timestamp) UnixFloat() float64 {
	return float64(t) / 1e9
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
	return t.Format("2006-01-02_15:04:05.000")
}
