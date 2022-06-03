/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var enablePGO, enablePropeller, enablePGOAndPropeller bool

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
		c := LoadConfig("FDO_settings.yaml")
		if enablePGO || enablePGOAndPropeller {
			buildInstrumented(c)
		}
		if enablePropeller {
			buildLabeled(c)
		}
		if enablePGOAndPropeller {
			buildLabeledOnPGO(c)
		}
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the project",
	Run: func(cmd *cobra.Command, args []string) {
		c := LoadConfig("FDO_settings.yaml")
		if enablePGO || enablePGOAndPropeller {
			testPGO(c)
		}
		if enablePropeller {
			testPropeller(c)
		}
		if enablePGOAndPropeller {
			// TODO: test PGO+Propeller
		}
	},
}

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

	rootCmd.PersistentFlags().BoolVarP(&enablePGO, "pgo", "", false, "enable pgo")
	rootCmd.PersistentFlags().BoolVarP(&enablePropeller, "propeller", "", false, "enable propeller")
	rootCmd.PersistentFlags().BoolVarP(&enablePGOAndPropeller, "pgo-and-propeller", "", false, "enable pgo and propeller")

	configCmd.DisableFlagParsing = true

	rootCmd.AddCommand(versionCmd, buildCmd, configCmd, testCmd)

}
