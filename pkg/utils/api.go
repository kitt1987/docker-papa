package utils

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

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

func WriteYaml(file string, obj interface{}) (err error) {
	bin, err := yaml.Marshal(obj)
	if err != nil {
		return
	}

	return ioutil.WriteFile(file, bin, 0644)
}

func ReadYaml(file string, obj interface{}) (err error) {
	bin, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}

	return yaml.Unmarshal(bin, obj)
}
