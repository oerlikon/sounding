package common

const (
	_   = iota
	KiB = 1 << (10 * iota)
	MiB
	GiB
	TiB
)

func ContainsString(a []string, s string) bool {
	for _, z := range a {
		if z == s {
			return true
		}
	}
	return false
}
