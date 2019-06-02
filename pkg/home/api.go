package home

type PaPaHome interface {
	WriteYaml(path string, yaml interface{}) error
	ReadYaml(path string, yaml interface{}) error
}

func Load() PaPaHome {
	return &papaHome{}
}
