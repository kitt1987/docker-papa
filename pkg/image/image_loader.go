package image

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/image"
	"github.com/docker/docker/layer"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/symlink"
	"github.com/golang/glog"
	"github.com/opencontainers/go-digest"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type imageLoader struct {
	imageDir  string
	manifests []manifestConf
}

func newDockerImageLoader(cc *dockerclient.Client, imageName string) (l *imageLoader, err error) {
	l = &imageLoader{}
	glog.V(3).Infof("Saving image %s", imageName)
	reader, err := cc.ImageSave(context.Background(), []string{imageName})
	if err != nil {
		glog.V(3).Infof("tar image failed: %s", err)
		return
	}

	defer reader.Close()

	l.imageDir, err = ioutil.TempDir("", "papa-imageName-")
	if err != nil {
		glog.V(3).Infof("tmpdir failed: %s", err)
		return
	}

	glog.V(3).Infof("Save image %s in tempdir %s", imageName, l.imageDir)
	if err = archive.Unpack(reader, l.imageDir, &archive.TarOptions{
		NoLchown:         true,
		IncludeSourceDir: true,
	}); err != nil {
		glog.V(3).Infof("untar image failed: %s", err)
		return
	}

	manifestPath, err := safePath(l.imageDir, manifestFileName)
	if err != nil {
		glog.V(3).Infof("manifest failed: %s", err)
		return
	}

	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		glog.V(3).Infof("open manifest failed at %s: %s", manifestPath, err)
		return
	}

	defer manifestFile.Close()

	var manifests []manifestItem
	if err = json.NewDecoder(manifestFile).Decode(&manifests); err != nil {
		glog.V(3).Infof("parse manifest failed at %s: %s", manifestPath, err)
		return
	}

	if len(manifests) == 0 {
		err = fmt.Errorf("no manifest found in the imageName")
		return
	}

	for i := range manifests {
		mc := manifestConf{
			manifest: &manifests[i],
		}

		var configPath string
		configPath, err = safePath(l.imageDir, mc.manifest.Config)
		if err != nil {
			glog.V(3).Infof("manifest configuration failed: %s", err)
			return
		}

		mc.rawConf, err = ioutil.ReadFile(configPath)
		if err != nil {
			glog.V(3).Infof("open manifest configuration failed at %s: %s", configPath, err)
			return
		}

		mc.conf, err = image.NewFromJSON(mc.rawConf)
		if err != nil {
			glog.V(3).Infof("parse manifest configuration failed at %s: %s", configPath, err)
			return
		}

		l.manifests = append(l.manifests, mc)
	}

	return
}

func (l *imageLoader) Close() error {
	return os.RemoveAll(l.imageDir)
}

const (
	manifestFileName = "manifest.json"
)

func safePath(base, path string) (string, error) {
	return symlink.FollowSymlinkInScope(filepath.Join(base, path), base)
}

type manifestItem struct {
	Config       string
	RepoTags     []string
	Layers       []string
	Parent       image.ID                                 `json:",omitempty"`
	LayerSources map[layer.DiffID]distribution.Descriptor `json:",omitempty"`
}

type manifestConf struct {
	manifest *manifestItem
	conf     *image.Image
	rawConf  []byte
}

func (l *imageLoader) GetLayers() (layers []*layerLoader, err error) {
	for _, m := range l.manifests {
		for i := range m.manifest.Layers {
			layerPath := m.manifest.Layers[i]
			layerID := m.conf.RootFS.DiffIDs[i]
			layerDir := filepath.Join(l.imageDir, filepath.Dir(layerPath))
			layerPath = filepath.Join(l.imageDir, layerPath)
			var fi os.FileInfo
			fi, err = os.Lstat(layerPath)
			if err != nil {
				glog.V(3).Infof("can't found layer at %s: %s", layerPath, err)
				return
			}

			layers = append(layers, &layerLoader{
				layerDir:  layerDir,
				layerPath: layerPath,
				descriptor: distribution.Descriptor{
					MediaType: schema2.MediaTypeUncompressedLayer,
					Size:      fi.Size(),
					Digest:    digest.Digest(layerID),
					Platform: &ociv1.Platform{
						Architecture: m.conf.Architecture,
						OS:           m.conf.OS,
					},
				},
			})
		}
	}

	return
}

func (l *imageLoader) GetManifests() (manifests []*manifestConf, err error) {
	return
}

type layerLoader struct {
	layerDir   string
	layerPath  string
	descriptor distribution.Descriptor
}

func (l *layerLoader) OpenReader() (r io.ReadCloser, err error) {
	return os.Open(l.layerPath)
}
