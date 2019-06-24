// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"github.com/kitt1987/docker-papa/pkg/container"
	"github.com/kitt1987/docker-papa/pkg/image"
	"github.com/spf13/cobra"
	"os"
)

// FIXME a command to show or purge legacy history

// containerCmd represents the container command
var containerCmd = &cobra.Command{
	Use:   "container",
	Short: "container",
	Long: `Recreate a container with the same arguments except a new image.
	docker-papa container -r turtle --image registry.cloudtogo.cn/cloudtogo.cn/official/turtle:1.7.0-build-create-new-cluster-20190520141436

	Show a docker command line could run the same container.
	docker-papa container -c turtle`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, containerArgs []string) {
		args.nameOrID = containerArgs[0]
		if actions.Recreate {
			if err := recreateContainer(); err != nil {
				fmt.Fprintf(os.Stderr, "container %s : %s\n", args.nameOrID, err)
				os.Exit(2)
			}
		}

		if actions.Parse {
			if cmd, err := parseContainer(); err != nil {
				fmt.Fprintf(os.Stderr, "container %s : %s\n", args.nameOrID, err)
				os.Exit(2)
			} else {
				fmt.Println(cmd)
			}
		}
	},
}

type containerActions struct {
	Recreate bool
	Parse    bool
	Recover bool //Get a legacy container back
}

type containerArgs struct {
	nameOrID   string
}

var (
	actions      containerActions
	args         containerArgs
	recreateOpts container.RecreateOptions
	cmd          string
)

func init() {
	rootCmd.AddCommand(containerCmd)

	// Here you will define your actions and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// containerCmd.PersistentFlags().String("foo", "", "A help for foo")

	containerCmd.Flags().StringVar(&recreateOpts.Image, "image", "",
		"Image for the container you specified by --id or --name if you would like to update it")
	containerCmd.Flags().BoolVar(&recreateOpts.RestartAlways, "restart-always", false,
		"Make the container you specified by --id or --name always restart on fail or boot")
	containerCmd.Flags().StringVar(&recreateOpts.Network, "net", "",
		"Change network of the container you specified by --id or --name")
	containerCmd.Flags().StringSliceVarP(&recreateOpts.Bindings, "volume", "l", recreateOpts.Bindings,
		"Mounts for the container you specified by --id or --name")
	containerCmd.Flags().BoolVar(&recreateOpts.RenewBindings, "renew-mounts", false,
		"Drop all mounts of the container")
	containerCmd.Flags().StringSliceVarP(&recreateOpts.Env, "env", "e", recreateOpts.Env,
		"Environment variables for the container you specified by --id or --name")
	containerCmd.Flags().BoolVar(&recreateOpts.RenewEnv, "renew-envs", false,
		"Drop all environment variables of the container")
	containerCmd.Flags().StringSliceVarP(&recreateOpts.PortMapping, "port", "p", recreateOpts.PortMapping,
		"Port mappings for the container you specified by --id or --name")
	containerCmd.Flags().BoolVar(&recreateOpts.RenewPortMapping, "renew-ports", false,
		"Drop all port mappings of the container")
	containerCmd.Flags().StringVar(&cmd, "cmd", "", "Command for the container")
	containerCmd.Flags().BoolVar(&recreateOpts.RenewCmd, "renew-cmd", false,
		"Drop all commands of the container")
	containerCmd.Flags().StringVar(&recreateOpts.Rename, "rename", "", "New name of the container")
	containerCmd.Flags().StringSliceVar(&recreateOpts.KeepFiles, "keep-file", recreateOpts.KeepFiles,
		"Keep files or directories after recreating")
	containerCmd.Flags().BoolVarP(&actions.Recreate, "recreate", "r", false,
		"Recreate a existed docker container with specified options. The current container will be renamed to" +
		"its original name with a suffix .legacy and stopped.")
	containerCmd.Flags().BoolVarP(&actions.Parse, "parse", "c", false,
		"Generate docker run command line from a existed container")
}

func recreateContainer() (err error) {
	c, err := container.GetExistedDockerContainer(args.nameOrID, dockerDaemonSocket)
	if err != nil {
		return
	}

	if len(recreateOpts.Image) > 0 {
		if found, err := image.ExistsLocally(recreateOpts.Image); err != nil || !found {
			if err = image.DockerPull(recreateOpts.Image); err != nil {
				fmt.Fprintf(os.Stderr, "can't pull image %s:%s. use local images instead.\n",
					recreateOpts.Image, err)
			}
		} else {
			fmt.Fprintf(os.Stdout, "Found image %s locally\n", recreateOpts.Image)
		}

		if len(cmd) > 0 {
			recreateOpts.Cmd = splitCliArgs(cmd)
		}

		if _, err = c.Recreate(&recreateOpts); err != nil {
			return
		}
	}

	return
}

func parseContainer() (cmd string, err error) {
	c, err := container.GetExistedDockerContainer(args.nameOrID, dockerDaemonSocket)
	if err != nil {
		return
	}

	cmd, err = c.ConvertToDockerCommand()
	return
}
