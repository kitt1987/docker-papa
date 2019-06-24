// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
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
	"github.com/kitt1987/docker-papa/pkg/ctx"
	"github.com/kitt1987/docker-papa/pkg/image"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args[0]) == 0 {
			fmt.Fprintf(os.Stderr, "image name is required\n")
			os.Exit(2)
		}

		context, _ := ctx.Current()
		var err error
		if strings.HasPrefix(args[0], context.RegistryName) {
			if len(context.Registry) == 0 {
				fmt.Fprintf(os.Stderr, "no registry specified in context %s\n", context.Name)
				os.Exit(2)
			}

			err = image.PushDirectly(args[0], context.Registry)
		} else {
			err = image.Push(args[0])
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "fail to push image %s: %s\n", args[0], err)
			os.Exit(2)
		}

		fmt.Printf("Image %s pushed", args[0])
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pushCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pushCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
