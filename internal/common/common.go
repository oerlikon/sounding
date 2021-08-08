package common

func ContainsString(a []string, s string) bool {
	for _, z := range a {
		if z == s {
			return true
		}
	}
	return false
}
