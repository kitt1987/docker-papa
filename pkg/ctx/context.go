package ctx

import (
	"github.com/kitt1987/docker-papa/pkg/home"
	"path"
)

const (
	ContextDir = "context.d"
)

type Context struct {
	Name         string `yaml:"name,omitempty"`
	Registry     string `yaml:"registry,omitempty"`
	RegistryName string `yaml:"registryName,omitempty"`
}

func (c Context) Save() (err error) {
	return home.Load().WriteYaml(path.Join(ContextDir, c.Name), &c)
}

func (c *Context) Load() (err error) {
	return home.Load().ReadYaml(path.Join(ContextDir, c.Name), c)
}
