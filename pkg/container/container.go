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
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type dockerContainer struct {
	ContainerInspectData types.ContainerJSON
	Cli                  *client.Client
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

	c = &dockerContainer{
		ContainerInspectData: containerInspectData,
		Cli:                  cli,
	}

	return
}

func (c *dockerContainer) Recreate(opts *RecreateOptions) (newID string, err error) {
	if len(opts.Image) > 0 {
		c.ContainerInspectData.Config.Image = opts.Image
	}

	if opts.RestartAlways {
		c.ContainerInspectData.HostConfig.RestartPolicy.Name = `aways`
	}

	if len(opts.Network) > 0 {
		c.ContainerInspectData.HostConfig.NetworkMode = container.NetworkMode(opts.Network)
	}

	if opts.RenewBindings {
		c.ContainerInspectData.HostConfig.Binds = []string{}
	}

	c.ContainerInspectData.HostConfig.Binds = append(c.ContainerInspectData.HostConfig.Binds, opts.Bindings...)
	mountParser := mounts.NewParser(runtime.GOOS)
	for _, bind := range opts.Bindings {
		var mountPoint *mounts.MountPoint
		mountPoint, err = mountParser.ParseMountRaw(bind, c.ContainerInspectData.HostConfig.VolumeDriver)
		if err != nil {
			err = fmt.Errorf("fail to parse bind %s cuz %s", bind, err)
			return
		}

		c.ContainerInspectData.HostConfig.Mounts = append(c.ContainerInspectData.HostConfig.Mounts, mount.Mount{
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
		c.ContainerInspectData.Config.Env = []string{}
	}

	c.ContainerInspectData.Config.Env = append(c.ContainerInspectData.Config.Env, opts.Env...)

	if opts.RenewPortMapping {
		c.ContainerInspectData.HostConfig.PortBindings = make(nat.PortMap)
	}

	_, bindings, err := nat.ParsePortSpecs(opts.PortMapping)
	for k, v := range bindings {
		c.ContainerInspectData.HostConfig.PortBindings[k] = v
	}

	if opts.RenewCmd {
		c.ContainerInspectData.Config.Cmd = strslice.StrSlice{}
	}

	c.ContainerInspectData.Config.Cmd = append(c.ContainerInspectData.Config.Cmd, opts.Cmd...)

	if len(opts.Rename) > 0 {
		c.ContainerInspectData.Name = opts.Rename
	}

	const tmpFilesToKeep = `.docker-files-to-keep`
	os.MkdirAll(tmpFilesToKeep, 0600)

	for _, fileToKeep := range opts.KeepFiles {
		replicator := func(f string) (err error) {
			fileHash := md5.Sum([]byte(f))
			dstFile := filepath.Join(tmpFilesToKeep, c.ContainerInspectData.ID+hex.EncodeToString(fileHash[:])+".tar")
			writer, err := os.Create(dstFile)
			if err != nil {
				return
			}

			defer writer.Close()
			reader, _, err := c.Cli.CopyFromContainer(context.Background(), c.ContainerInspectData.ID, f)
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
	_, err = c.Cli.ContainerUpdate(ctx, c.ContainerInspectData.ID, container.UpdateConfig{
		RestartPolicy: container.RestartPolicy{
			Name: "no",
		}})
	if err != nil {
		return
	}

	fmt.Fprintln(os.Stdout, "Mark container", c.ContainerInspectData.Name, " as non-autorestart")

	if err = c.Cli.ContainerStop(ctx, c.ContainerInspectData.ID, nil); err != nil {
		return
	}

	fmt.Fprintln(os.Stdout, "Stop container", c.ContainerInspectData.Name)

	err = c.Cli.ContainerRename(ctx, c.ContainerInspectData.ID, c.ContainerInspectData.Name+".legacy")
	if err != nil {
		return
	}

	fmt.Fprintf(os.Stdout, "Rename container %s to %s\n", c.ContainerInspectData.Name,
		c.ContainerInspectData.Name+".legacy")

	created, err := c.Cli.ContainerCreate(ctx, c.ContainerInspectData.Config, c.ContainerInspectData.HostConfig,
		&network.NetworkingConfig{
			EndpointsConfig: c.ContainerInspectData.NetworkSettings.Networks,
		},
		c.ContainerInspectData.Name)
	if err != nil {
		return
	}

	fmt.Fprintln(os.Stdout, "Create new container", c.ContainerInspectData.Name)

	for _, fileToKeep := range opts.KeepFiles {
		replicator := func(f string) (err error) {
			fileHash := md5.Sum([]byte(f))
			dstFile := filepath.Join(tmpFilesToKeep, c.ContainerInspectData.ID+hex.EncodeToString(fileHash[:])+".tar")
			reader, err := os.Open(dstFile)
			if err != nil {
				return
			}

			defer reader.Close()
			err = c.Cli.CopyToContainer(context.Background(), created.ID, fileToKeep, reader, types.CopyToContainerOptions{})
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
	if err = c.Cli.ContainerStart(ctx, newID, types.ContainerStartOptions{}); err != nil {
		return
	}

	fmt.Fprintln(os.Stdout, "Start container", c.ContainerInspectData.Name)

	return
}

func (c *dockerContainer) ConvertToDockerCommand() (cmd string, err error) {
	cmdArray := []string{`docker run`}
	cmdArray = append(cmdArray, `--name`, c.ContainerInspectData.Name[1:])
	if c.ContainerInspectData.HostConfig.RestartPolicy.MaximumRetryCount > 0 {
		cmdArray = append(cmdArray, `--restart`,
			fmt.Sprintf(`%s:%d`, c.ContainerInspectData.HostConfig.RestartPolicy.Name,
				c.ContainerInspectData.HostConfig.RestartPolicy.MaximumRetryCount))
	} else {
		cmdArray = append(cmdArray, `--restart`, c.ContainerInspectData.HostConfig.RestartPolicy.Name)
	}

	if c.ContainerInspectData.HostConfig.AutoRemove {
		cmdArray = append(cmdArray, `--rm`)
	}

	if !c.ContainerInspectData.Config.AttachStdin && !c.ContainerInspectData.Config.AttachStdout &&
		!c.ContainerInspectData.Config.AttachStderr {
		cmdArray = append(cmdArray, `-d`)
	} else {
		if c.ContainerInspectData.Config.AttachStdin {
			cmdArray = append(cmdArray, `-a stdin`)
		}

		if c.ContainerInspectData.Config.AttachStdout {
			cmdArray = append(cmdArray, `-a stdout`)
		}

		if c.ContainerInspectData.Config.AttachStderr {
			cmdArray = append(cmdArray, `-a stderr`)
		}
	}

	if c.ContainerInspectData.Config.Tty {
		cmdArray = append(cmdArray, `-t`)
	}

	for port, binding := range c.ContainerInspectData.HostConfig.PortBindings {
		for _, hostPort := range binding {
			if len(hostPort.HostIP) > 0 {
				cmdArray = append(cmdArray, fmt.Sprintf(`-p %s:%s:%s`, port, hostPort.HostIP, hostPort.HostPort))
			} else {
				cmdArray = append(cmdArray, fmt.Sprintf(`-p %s:%s`, port, hostPort.HostPort))
			}
		}
	}

	if c.ContainerInspectData.HostConfig.NetworkMode.IsNone() {
		cmdArray = append(cmdArray, `--net=none`)
	}

	if c.ContainerInspectData.HostConfig.NetworkMode.IsHost() {
		cmdArray = append(cmdArray, `--net=host`)
	}

	if c.ContainerInspectData.HostConfig.NetworkMode.IsUserDefined() {
		cmdArray = append(cmdArray, `--net=`+c.ContainerInspectData.HostConfig.NetworkMode.UserDefined())
	}

	for _, dns := range c.ContainerInspectData.HostConfig.DNS {
		cmdArray = append(cmdArray, `--dns`, dns)
	}

	for _, dns := range c.ContainerInspectData.HostConfig.DNSOptions {
		cmdArray = append(cmdArray, `--dns-option`, dns)
	}

	for _, dns := range c.ContainerInspectData.HostConfig.DNSSearch {
		cmdArray = append(cmdArray, `--dns-search`, dns)
	}

	for _, host := range c.ContainerInspectData.HostConfig.ExtraHosts {
		cmdArray = append(cmdArray, `--add-host`, host)
	}

	if c.ContainerInspectData.HostConfig.Privileged {
		cmdArray = append(cmdArray, `--privileged`)
	}

	if c.ContainerInspectData.HostConfig.PublishAllPorts {
		cmdArray = append(cmdArray, `-P`)
	}

	for _, containerMount := range c.ContainerInspectData.Mounts {
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

	for _, env := range c.ContainerInspectData.Config.Env {
		if strings.HasPrefix(env, `PATH`) {
			continue
		}

		cmdArray = append(cmdArray, `-e`, env)
	}

	if len(c.ContainerInspectData.Config.Entrypoint) > 0 {
		cmdArray = append(cmdArray, fmt.Sprintf(`--entrypoint='%s'`,
			strings.Join(c.ContainerInspectData.Config.Entrypoint, ` `)))
	}

	if c.ContainerInspectData.Config.Healthcheck != nil {
		cmdArray = append(cmdArray, fmt.Sprintf(`--health-cmd='%s'`,
			strings.Join(c.ContainerInspectData.Config.Healthcheck.Test, ` `)))
		cmdArray = append(cmdArray, `--health-interval=`,
			c.ContainerInspectData.Config.Healthcheck.Interval.String())
		cmdArray = append(cmdArray, fmt.Sprintf(`--health-retries=%d`,
			c.ContainerInspectData.Config.Healthcheck.Retries))
		cmdArray = append(cmdArray, `--health-timeout=`,
			c.ContainerInspectData.Config.Healthcheck.Timeout.String())
		cmdArray = append(cmdArray, `--health-start-period=`,
			c.ContainerInspectData.Config.Healthcheck.StartPeriod.String())
	}

	cmdArray = append(cmdArray, c.ContainerInspectData.Config.Image)
	cmdArray = append(cmdArray, strings.Join(c.ContainerInspectData.Args, ` `))
	cmd = strings.Join(cmdArray, ` `)
	return
}
