package image

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"os"
)

func Pull(image string) (err error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.29"))
	if err != nil {
		return
	}

	resp, err := cli.ImagePull(context.Background(), image, types.ImagePullOptions{})
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

func ExistsLocally(image string) (yes bool, err error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.29"))
	if err != nil {
		return
	}

	matched, err := cli.ImageList(context.Background(), types.ImageListOptions{
		All: true,
		Filters: filters.NewArgs(filters.Arg("reference", image)),
	})

	if err != nil {
		return
	}

	yes = len(matched) > 0
	return
}