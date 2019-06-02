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
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// contextCmd represents the context command
var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "A context of pushing or pulling images",
	Long: `Manipulate context of docker-papa. 

Samples:
  Create a new context,
  docker-papa context create uat --registry registry-bj.uat.abc.cn
  
  Switch to another context,
  docker-papa context switch uat`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		action := strings.ToLower(args[0])
		args = args[1:]
		switch action {
		case "create":
			if len(args) == 0 || len(args[0]) == 0 {
				fmt.Fprintf(os.Stderr, "context name is required\n")
				os.Exit(2)
			}

			ctxArgs.name = args[0]

			if len(ctxArgs.registry) == 0 {
				fmt.Fprintf(os.Stderr, "a registry is required to bind to registry.context\n")
				os.Exit(2)
			}

			c := ctx.Context{
				Name:         args[0],
				Registry:     ctxArgs.registry,
				RegistryName: ctxArgs.registryName,
			}

			err := ctx.Create(&c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "save context failed: %s\n", err)
				os.Exit(2)
			}

			err = ctx.Switch(&c, ctx.SingleUserContext)
			if err != nil {
				fmt.Fprintf(os.Stderr, "switch context failed: %s\n", err)
				os.Exit(2)
			}

			fmt.Printf("context %s is used\n", c.Name)

		case "switch":
			if len(args) == 0 || len(args[0]) == 0 {
				fmt.Fprintf(os.Stderr, "context name is required\n")
				os.Exit(2)
			}

			c, err := ctx.Load(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "load context %s failed: %s\n", args[0], err)
				os.Exit(2)
			}

			err = ctx.Switch(c, ctx.SingleUserContext)
			if err != nil {
				fmt.Fprintf(os.Stderr, "switch context failed: %s\n", err)
				os.Exit(2)
			}

			fmt.Printf("context %s is used\n", c.Name)

		case "list":
		case "purge":
		default:
			fmt.Printf("context called with %#v/n", args)
		}
	},
}

type contextActions struct {
	Create bool
	List   bool
	Purge  bool
}

type contextArgs struct {
	name         string
	registryName string
	registry     string
}

var (
	ctxActions contextActions
	ctxArgs    contextArgs
)

func init() {
	rootCmd.AddCommand(contextCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// contextCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// contextCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	contextCmd.Flags().StringVar(&ctxArgs.registry, "registry", "",
		"The default registry in the context to which traffic will be forwarded when pushing to or pulling "+
			"from the well-known registry in the context")
	contextCmd.Flags().StringVar(&ctxArgs.registryName, "well-known-registry-in-context", "registry.context",
		"This argument shouldn't be changed unless it occupies some domain already been used.")
}
