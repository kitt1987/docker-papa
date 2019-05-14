package utils

func SliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func Diff(from, to []string) (d []string) {
	if len(to) == 0 {
		return from
	}

	toMap := make(map[string]struct{})
	for i := range to {
		toMap[to[i]] = struct{}{}
	}

	for i := range from {
		if _, foundInTo := toMap[from[i]]; !foundInTo {
			d = append(d, from[i])
		}
	}

	return
}