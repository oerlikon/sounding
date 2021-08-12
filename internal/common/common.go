package common

const (
	_   = iota
	KiB = 1 << (10 * iota)
	MiB
	GiB
	TiB
)

func FindString(a []string, s string) int {
	for i, z := range a {
		if z == s {
			return i
		}
	}
	return -1
}
