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
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var dockerDaemonSocket string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "docker-papa",
	Short: "A Docker client better than official",
}

// Execute adds all child commands to the root command and sets actions appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if len(os.Args) > 1 {
		if _, _, err := rootCmd.Find(os.Args[1:]); err != nil {
			fmt.Println(err)
			dockerPath, err := exec.LookPath("docker")
			if err != nil {
				fmt.Println(err)
				os.Exit(2)
			}

			// FIXME test on dumb Windows
			err = syscall.Exec(dockerPath, os.Args, os.Environ())
			if err != nil {
				fmt.Println(err)
				os.Exit(2)
			}
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your actions and configuration settings.
	// Cobra supports persistent actions, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.docker-papa.yaml)")
	rootCmd.PersistentFlags().StringVarP(&dockerDaemonSocket, "host", "H", "", "Daemon socket to connect to")
	flag.CommandLine.Parse([]string{})
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// Cobra also supports local actions, which will only run
	// when this action is called directly.
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".docker-papa" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".docker-papa")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
