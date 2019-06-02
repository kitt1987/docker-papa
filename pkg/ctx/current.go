package ctx

import (
	"fmt"
	"github.com/kitt1987/docker-papa/pkg/home"
	"github.com/kitt1987/docker-papa/pkg/utils"
	"os"
	"path"
)

type ContextMode string

const (
	SingleUserContext        ContextMode = "single-user"
	MultipleUserContext      ContextMode = "multi-user"
	CurrentContextFile                   = "context"
	GlobalCurrentContextFile             = "papa-context"
)

type CurrentContext struct {
	*Context `yaml:"inline"`
	CtxMode ContextMode `yaml:"contextMode,omitempty"`
}

func (c CurrentContext) Save() (err error) {
	switch c.CtxMode {
	case MultipleUserContext:
		err = utils.WriteYaml(getCtxFilePathInMultiUserMode(), &c)
		if err != nil {
			return
		}

		fallthrough
	case SingleUserContext:
		return home.Load().WriteYaml(CurrentContextFile, &c)
	default:
		err = fmt.Errorf("only single-user or multi-user is supported")
	}

	return
}

func (c *CurrentContext) Load() (err error) {
	err = home.Load().ReadYaml(CurrentContextFile, c)
	if err != nil {
		return
	}

	if c.CtxMode != MultipleUserContext {
		return
	}

	return utils.ReadYaml(getCtxFilePathInMultiUserMode(), c)
}

func getCtxFilePathInMultiUserMode() string {
	return path.Join(os.TempDir(), fmt.Sprintf("%s.%d", GlobalCurrentContextFile, os.Getppid()))
}
