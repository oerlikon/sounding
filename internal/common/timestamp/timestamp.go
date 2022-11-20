package timestamp

import "time"

type T int64

func Stamp(t time.Time) T {
	if t.IsZero() {
		return 0
	}
	return T(t.UnixNano())
}

func Float(s float64) T {
	return T(s * 1e9)
}

func Milli(ms int64) T {
	return T(ms * 1e6)
}

func Micro(us int64) T {
	return T(us * 1e3)
}

func (t T) Time() time.Time {
	return time.Unix(0, int64(t)).UTC()
}

func (t T) Unix() int {
	return int(t / 1e9)
}

func (t T) UnixFloat() float64 {
	return float64(t) / 1e9
}

func (t T) UnixMilli() int64 {
	return int64(t / 1e6)
}

func (t T) UnixMicro() int64 {
	return int64(t / 1e3)
}

func (t T) Add(d time.Duration) T {
	return T(int64(t) + int64(d))
}

func (t T) Sub(u T) time.Duration {
	return time.Duration(t - u)
}

func (t T) Truncate(d time.Duration) T {
	return T(int64(t) - int64(t)%int64(d))
}

func (t T) Format(layout string) string {
	return t.Time().Format(layout)
}

func (t T) S() string {
	if t == 0 {
		return "0"
	}
	return t.Format("2006-01-02_15:04:05.000")
}
