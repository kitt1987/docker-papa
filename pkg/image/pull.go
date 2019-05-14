package image

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"os"
)

func Pull(image string) (err error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.29"))
	if err != nil {
		return
	}

	reader, err := cli.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		return
	}

	defer reader.Close()
	io.Copy(os.Stdout, reader)
	return
}
