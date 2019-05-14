package container

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/docker/volume/mounts"
	"github.com/docker/go-connections/nat"
	"github.com/kitt1987/docker-papa/pkg/utils"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type dockerContainer struct {
	containerInspectData types.ContainerJSON
	imageInspectData     types.ImageInspect
	cli                  *client.Client
}

func GetExistedDockerContainer(IDorName, daemon string) (c DockerContainer, err error) {
	var cli *client.Client
	if len(daemon) > 0 {
		cli, err = client.NewClientWithOpts(client.WithHost(daemon), client.WithVersion("1.29"))
	} else {
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.29"))
	}

	if err != nil {
		return
	}

	if len(IDorName) == 0 {
		err = fmt.Errorf("container ID or name is required")
		return
	}

	idFilter := filters.NewArgs()
	idFilter.Add("id", IDorName)
	nameFilter := filters.NewArgs()
	nameFilter.Add("name", IDorName)

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: nameFilter,
	})

	if err != nil {
		return
	}

	if len(containers) == 0 {
		containers, err = cli.ContainerList(ctx, types.ContainerListOptions{
			All:     true,
			Filters: idFilter,
		})
	}

	if err != nil {
		return
	}

	if len(containers) == 0 {
		err = fmt.Errorf("no container found")
		return
	}

	if len(containers) > 1 {
		var names []string
		for _, c := range containers {
			names = append(names, c.Names...)
		}

		err = fmt.Errorf("more than 1 containers matched name/id %s where are %s", IDorName,
			strings.Join(names, ","))
		return
	}

	containerInspectData, err := cli.ContainerInspect(ctx, containers[0].ID)
	if err != nil {
		return
	}

	imageInspectData, _, err := cli.ImageInspectWithRaw(ctx, containerInspectData.Image)
	if err != nil {
		return
	}

	c = &dockerContainer{
		containerInspectData: containerInspectData,
		imageInspectData:     imageInspectData,
		cli:                  cli,
	}

	return
}

func (c *dockerContainer) Recreate(opts *RecreateOptions) (newID string, err error) {
	if len(opts.Image) > 0 {
		c.containerInspectData.Config.Image = opts.Image
	}

	if opts.RestartAlways {
		c.containerInspectData.HostConfig.RestartPolicy.Name = `aways`
	}

	if len(opts.Network) > 0 {
		c.containerInspectData.HostConfig.NetworkMode = container.NetworkMode(opts.Network)
	}

	if opts.RenewBindings {
		c.containerInspectData.HostConfig.Binds = []string{}
	}

	c.containerInspectData.HostConfig.Binds = append(c.containerInspectData.HostConfig.Binds, opts.Bindings...)
	mountParser := mounts.NewParser(runtime.GOOS)
	for _, bind := range opts.Bindings {
		var mountPoint *mounts.MountPoint
		mountPoint, err = mountParser.ParseMountRaw(bind, c.containerInspectData.HostConfig.VolumeDriver)
		if err != nil {
			err = fmt.Errorf("fail to parse bind %s cuz %s", bind, err)
			return
		}

		c.containerInspectData.HostConfig.Mounts = append(c.containerInspectData.HostConfig.Mounts, mount.Mount{
			Type:        mount.TypeBind,
			Source:      mountPoint.Source,
			Target:      mountPoint.Destination,
			ReadOnly:    !mountPoint.RW,
			Consistency: mount.ConsistencyFull,
			BindOptions: &mount.BindOptions{
				Propagation: mountPoint.Propagation,
			},
		})
	}

	if opts.RenewEnv {
		c.containerInspectData.Config.Env = []string{}
	}

	c.containerInspectData.Config.Env = append(c.containerInspectData.Config.Env, opts.Env...)

	if opts.RenewPortMapping {
		c.containerInspectData.HostConfig.PortBindings = make(nat.PortMap)
	}

	_, bindings, err := nat.ParsePortSpecs(opts.PortMapping)
	for k, v := range bindings {
		c.containerInspectData.HostConfig.PortBindings[k] = v
	}

	if opts.RenewCmd {
		c.containerInspectData.Config.Cmd = strslice.StrSlice{}
	}

	c.containerInspectData.Config.Cmd = append(c.containerInspectData.Config.Cmd, opts.Cmd...)

	if len(opts.Rename) > 0 {
		c.containerInspectData.Name = opts.Rename
	}

	const tmpFilesToKeep = `.docker-files-to-keep`
	os.MkdirAll(tmpFilesToKeep, 0600)

	for _, fileToKeep := range opts.KeepFiles {
		replicator := func(f string) (err error) {
			fileHash := md5.Sum([]byte(f))
			dstFile := filepath.Join(tmpFilesToKeep, c.containerInspectData.ID+hex.EncodeToString(fileHash[:])+".tar")
			writer, err := os.Create(dstFile)
			if err != nil {
				return
			}

			defer writer.Close()
			reader, _, err := c.cli.CopyFromContainer(context.Background(), c.containerInspectData.ID, f)
			if err != nil {
				return
			}

			defer reader.Close()

			if _, err = io.Copy(writer, reader); err != nil {
				return
			}

			return
		}

		if err = replicator(fileToKeep); err != nil {
			return
		}
	}

	ctx := context.Background()
	_, err = c.cli.ContainerUpdate(ctx, c.containerInspectData.ID, container.UpdateConfig{
		RestartPolicy: container.RestartPolicy{
			Name: "no",
		}})
	if err != nil {
		return
	}

	fmt.Fprintln(os.Stdout, "Mark container", c.containerInspectData.Name, " as non-autorestart")

	if err = c.cli.ContainerStop(ctx, c.containerInspectData.ID, nil); err != nil {
		return
	}

	fmt.Fprintln(os.Stdout, "Stop container", c.containerInspectData.Name)

	err = c.cli.ContainerRename(ctx, c.containerInspectData.ID, c.containerInspectData.Name+".legacy")
	if err != nil {
		return
	}

	fmt.Fprintf(os.Stdout, "Rename container %s to %s\n", c.containerInspectData.Name,
		c.containerInspectData.Name+".legacy")

	created, err := c.cli.ContainerCreate(ctx, c.containerInspectData.Config, c.containerInspectData.HostConfig,
		&network.NetworkingConfig{
			EndpointsConfig: c.containerInspectData.NetworkSettings.Networks,
		},
		c.containerInspectData.Name)
	if err != nil {
		return
	}

	fmt.Fprintln(os.Stdout, "Create new container", c.containerInspectData.Name)

	for _, fileToKeep := range opts.KeepFiles {
		replicator := func(f string) (err error) {
			fileHash := md5.Sum([]byte(f))
			dstFile := filepath.Join(tmpFilesToKeep, c.containerInspectData.ID+hex.EncodeToString(fileHash[:])+".tar")
			reader, err := os.Open(dstFile)
			if err != nil {
				return
			}

			defer reader.Close()
			err = c.cli.CopyToContainer(context.Background(), created.ID, fileToKeep, reader, types.CopyToContainerOptions{})
			if err != nil {
				return
			}

			return
		}

		if err = replicator(fileToKeep); err != nil {
			return
		}
	}

	newID = created.ID
	if err = c.cli.ContainerStart(ctx, newID, types.ContainerStartOptions{}); err != nil {
		return
	}

	fmt.Fprintln(os.Stdout, "Start container", c.containerInspectData.Name)

	return
}

func (c *dockerContainer) ConvertToDockerCommand() (cmd string, err error) {
	cmdArray := []string{`docker run`}
	cmdArray = append(cmdArray, `--name`, c.containerInspectData.Name[1:])
	if c.containerInspectData.HostConfig.RestartPolicy.MaximumRetryCount > 0 {
		cmdArray = append(cmdArray, `--restart`,
			fmt.Sprintf(`%s:%d`, c.containerInspectData.HostConfig.RestartPolicy.Name,
				c.containerInspectData.HostConfig.RestartPolicy.MaximumRetryCount))
	} else {
		cmdArray = append(cmdArray, `--restart`, c.containerInspectData.HostConfig.RestartPolicy.Name)
	}

	if c.containerInspectData.HostConfig.AutoRemove {
		cmdArray = append(cmdArray, `--rm`)
	}

	if !c.containerInspectData.Config.AttachStdin && !c.containerInspectData.Config.AttachStdout &&
		!c.containerInspectData.Config.AttachStderr {
		cmdArray = append(cmdArray, `-d`)
	} else {
		if c.containerInspectData.Config.AttachStdin {
			cmdArray = append(cmdArray, `-a stdin`)
		}

		if c.containerInspectData.Config.AttachStdout {
			cmdArray = append(cmdArray, `-a stdout`)
		}

		if c.containerInspectData.Config.AttachStderr {
			cmdArray = append(cmdArray, `-a stderr`)
		}
	}

	if c.containerInspectData.Config.Tty {
		cmdArray = append(cmdArray, `-t`)
	}

	for port, binding := range c.containerInspectData.HostConfig.PortBindings {
		for _, hostPort := range binding {
			if len(hostPort.HostIP) > 0 {
				cmdArray = append(cmdArray, fmt.Sprintf(`-p %s:%s:%s`, hostPort.HostIP, hostPort.HostPort, port))
			} else {
				cmdArray = append(cmdArray, fmt.Sprintf(`-p %s:%s`, hostPort.HostPort, port))
			}
		}
	}

	if c.containerInspectData.HostConfig.NetworkMode.IsNone() {
		cmdArray = append(cmdArray, `--net=none`)
	}

	if c.containerInspectData.HostConfig.NetworkMode.IsHost() {
		cmdArray = append(cmdArray, `--net=host`)
	}

	if c.containerInspectData.HostConfig.NetworkMode.IsUserDefined() {
		cmdArray = append(cmdArray, `--net=`+c.containerInspectData.HostConfig.NetworkMode.UserDefined())
	}

	for _, dns := range c.containerInspectData.HostConfig.DNS {
		cmdArray = append(cmdArray, `--dns`, dns)
	}

	for _, dns := range c.containerInspectData.HostConfig.DNSOptions {
		cmdArray = append(cmdArray, `--dns-option`, dns)
	}

	for _, dns := range c.containerInspectData.HostConfig.DNSSearch {
		cmdArray = append(cmdArray, `--dns-search`, dns)
	}

	for _, host := range c.containerInspectData.HostConfig.ExtraHosts {
		cmdArray = append(cmdArray, `--add-host`, host)
	}

	if c.containerInspectData.HostConfig.Privileged {
		cmdArray = append(cmdArray, `--privileged`)
	}

	if c.containerInspectData.HostConfig.PublishAllPorts {
		cmdArray = append(cmdArray, `-P`)
	}

	for _, containerMount := range c.containerInspectData.Mounts {
		if containerMount.Type != mount.TypeBind {
			fmt.Fprintf(os.Stdout, "volume %s:%s with type %s is ignored", containerMount.Source,
				containerMount.Destination, containerMount.Type)
			continue
		}

		volumeOpts := containerMount.Source + `:` + containerMount.Destination
		var opts []string
		if !containerMount.RW {
			opts = append(opts, `ro`)
		}

		if len(containerMount.Propagation) > 0 {
			opts = append(opts, string(containerMount.Propagation))
		}

		if len(opts) > 0 {
			volumeOpts = volumeOpts + `:` + strings.Join(opts, `,`)
		}

		cmdArray = append(cmdArray, `-v`, volumeOpts)
	}

	envs := utils.Diff(c.containerInspectData.Config.Env, c.imageInspectData.Config.Env)
	for _, env := range envs {
		cmdArray = append(cmdArray, `-e`, env)
	}

	if len(c.containerInspectData.Config.Entrypoint) > 0 &&
		!utils.SliceEqual(
			c.containerInspectData.Config.Entrypoint,
			c.imageInspectData.ContainerConfig.Entrypoint,
		) {
		cmdArray = append(cmdArray, fmt.Sprintf(`--entrypoint='%s'`,
			strings.Join(c.containerInspectData.Config.Entrypoint, ` `)))
	}

	if c.containerInspectData.Config.Healthcheck != nil {
		cmdArray = append(cmdArray, fmt.Sprintf(`--health-cmd='%s'`,
			strings.Join(c.containerInspectData.Config.Healthcheck.Test, ` `)))
		cmdArray = append(cmdArray, `--health-interval=`,
			c.containerInspectData.Config.Healthcheck.Interval.String())
		cmdArray = append(cmdArray, fmt.Sprintf(`--health-retries=%d`,
			c.containerInspectData.Config.Healthcheck.Retries))
		cmdArray = append(cmdArray, `--health-timeout=`,
			c.containerInspectData.Config.Healthcheck.Timeout.String())
		cmdArray = append(cmdArray, `--health-start-period=`,
			c.containerInspectData.Config.Healthcheck.StartPeriod.String())
	}

	cmdArray = append(cmdArray, c.containerInspectData.Config.Image)
	cmdArray = append(cmdArray, strings.Join(c.containerInspectData.Config.Cmd, ` `))
	cmd = strings.Join(cmdArray, ` `)
	return
}
