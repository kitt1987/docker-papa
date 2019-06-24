package image

import (
	"context"
	"fmt"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	registryclient "github.com/docker/distribution/registry/client"
	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/golang/glog"
	"github.com/opencontainers/go-digest"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func Push(image string) (err error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithVersion("1.29"))
	if err != nil {
		return
	}

	resp, err := cli.ImagePush(context.Background(), image, types.ImagePushOptions{})
	if err != nil {
		return
	}

	defer resp.Close()
	fd, isTerminal := term.GetFdInfo(os.Stdout)
	if err := jsonmessage.DisplayJSONMessagesStream(resp, os.Stdout, fd, isTerminal, nil); err != nil {
		return err
	}
	return
}

func PushDirectly(image, remote string) (err error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithVersion("1.29"))
	if err != nil {
		return
	}

	imageLoader, err := newDockerImageLoader(cli, image)
	if err != nil {
		glog.V(3).Infof("open image %s: %s", image, err)
		return
	}

	defer imageLoader.Close()

	repoName, err := reference.ParseNamed(image)
	if err != nil {
		glog.V(3).Infof("parse image %s failed: %s", image, err)
		return
	}

	repo := reference.Path(repoName)
	repoName, err = reference.WithName(repo)
	if err != nil {
		glog.V(3).Infof("parse image %s failed: %s", repo, err)
		return
	}

	glog.V(3).Infof("prepare to push image %s to %s", image, remote)
	var baseURLs []string
	if strings.HasPrefix(remote, "http") {
		baseURLs = append(baseURLs, remote)
	} else {
		baseURLs = append(baseURLs, "http://"+remote, "https://"+remote)
	}

	var repoService distribution.Repository
	for _, baseURL := range baseURLs {
		glog.V(3).Infof("open registry %s", baseURL)
		repoService, err = registryclient.NewRepository(repoName, baseURL, http.DefaultTransport)
		if err != nil {
			glog.V(3).Infof("open repository %s failed: %s", repoName, err)
			continue
		}

		break
	}

	if repoService == nil {
		return
	}

	netCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	maniService, err := repoService.Manifests(netCtx)
	if err != nil {
		glog.V(3).Infof("open manifest service failed: %s", err)
		return
	}

	blobStore := repoService.Blobs(netCtx)

	layers, err := imageLoader.GetLayers()
	if err != nil {
		glog.V(3).Infof("read local layers failed: %s", err)
		return
	}

	for _, layer := range layers {
		if _, err = blobStore.Stat(netCtx, layer.descriptor.Digest); err != nil {
			var bw distribution.BlobWriter
			if bw, err = blobStore.Create(netCtx); err != nil {
				glog.V(3).Infof("create blobs failed: %s", err)
				return
			}

			var reader io.ReadCloser
			if reader, err = layer.OpenReader(); err != nil {
				glog.V(3).Infof("read local layers failed: %s", err)
				return
			}

			if _, err = bw.ReadFrom(reader); err != nil {
				glog.V(3).Infof("upload blobs failed: %s", err)
				reader.Close()
				return
			}

			reader.Close()

			if _, err = bw.Commit(netCtx, layer.descriptor); err != nil {
				glog.V(3).Infof("commit blobs failed: %s", err)
				return
			}
		}
	}

	manifests, err := imageLoader.GetManifests()
	if err != nil {
		glog.V(3).Infof("read local manifest failed: %s", err)
		return
	}

	for _, m := range manifests {
		dgst, err := digest.Parse(m.conf.Config.Image)
		if err != nil {
			glog.V(3).Infof("read local manifest digest failed: %s", err)
			return err
		}

		if existed, _ := maniService.Exists(netCtx, dgst); existed {
			glog.V(3).Infof("ignore existed manifest %s", dgst)
			continue
		}

		builder := schema2.NewManifestBuilder(blobStore, schema2.MediaTypeManifest, m.rawConf)
		manifest, err := builder.Build(netCtx)
		if err != nil {
			glog.V(3).Infof("build local manifest failed: %s", err)
			return err
		}

		newDgst, err := maniService.Put(netCtx, manifest)
		if err != nil {
			glog.V(3).Infof("put manifest failed: %s", err)
			return err
		}

		fmt.Println("Digest of the new image is", newDgst.String())
	}

	return
}
