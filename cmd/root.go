/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "FDO",
	Short: "FDO(Feedback Directed Optimizer) is a tool for optimizing CMake project using clang and propeller",
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// run default build all command
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of FDO",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Feedback directed optimizer v0.1")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "config CMake arguments",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var source_dir = args[0]
		ConfigDir(source_dir, args[1:])
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the project",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cmake " + strings.Join(args, " "))
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the project",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("config" + strings.Join(args, " "))
	},
}

var sourceDir string

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.FDO.yaml)")
	configCmd.DisableFlagParsing = true

	rootCmd.AddCommand(versionCmd, buildCmd, configCmd, testCmd)

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
