package home

import (
	"fmt"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/kitt1987/docker-papa/pkg/utils"
	"github.com/mitchellh/go-homedir"
	"path"
)

type papaHome struct {
}

func (h *papaHome) WriteYaml(file string, obj interface{}) (err error) {
	fileName := path.Base(file)
	if fileName == "." || fileName == "/" {
		err = fmt.Errorf("not a file path: %s", file)
		return
	}

	filePath, err := h.AssureParentDir(file)
	if err != nil {
		return
	}

	return utils.WriteYaml(path.Join(filePath, fileName), obj)
}

func (h *papaHome) ReadYaml(file string, obj interface{}) (err error) {
	fileName := path.Base(file)
	if fileName == "." || fileName == "/" {
		err = fmt.Errorf("not a file path: %s", file)
		return
	}

	filePath, err := h.AssureParentDir(file)
	if err != nil {
		return
	}

	return utils.ReadYaml(path.Join(filePath, fileName), obj)
}

func (h *papaHome) AssureParentDir(file string) (filePath string, err error) {
	homePath, err := getHomePath()
	if err != nil {
		return
	}

	filePath = path.Join(homePath, path.Dir(file))
	err = fileutils.CreateIfNotExists(filePath, true)
	return
}

func getHomePath() (homePath string, err error) {
	userHome, err := homedir.Dir()
	if err != nil {
		return
	}

	homePath = path.Join(userHome, ".papa")
	err = fileutils.CreateIfNotExists(homePath, true)
	if err != nil {
		return
	}

	return
}

func getCtxHome() (home string, err error) {
	userHome, err := homedir.Dir()
	if err != nil {
		return
	}

	home = path.Join(userHome, ".papa", "ctx")
	err = fileutils.CreateIfNotExists(home, true)
	if err != nil {
		return
	}

	return
}
