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

var jobs int
var target, config, test_settings string

func LoadSetings() (Config, TestScript) {
	c := LoadConfig("FDO_settings.yaml")
	var t TestScript
	if test_settings == "" {
		t = LoadTestScript(c.Source + "/FDO_test.yaml")
	} else {
		t = LoadTestScript(test_settings)
	}
	return c, t
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the project",
	Run: func(cmd *cobra.Command, args []string) {
		c, t := LoadSetings()
		if enablePGO || enablePGOAndPropeller {
			buildInstrumented(c, t)
		}
		if enablePropeller {
			buildLabeled(c, t)
		}
		if enablePGOAndPropeller {
			buildLabeledOnPGO(c, t)
		}
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the project",
	Run: func(cmd *cobra.Command, args []string) {
		c, t := LoadSetings()
		if enablePGO || enablePGOAndPropeller {
			testPGO(c, t)
		}
		if enablePropeller {
			testPropeller(c, t)
		}
		if enablePGOAndPropeller {
			testPGOAndPropeller(c, t)
		}
	},
}

var optCmd = &cobra.Command{
	Use:   "opt",
	Short: "Optimize the project",
	Run: func(cmd *cobra.Command, args []string) {
		c, t := LoadSetings()
		if enablePGO || enablePGOAndPropeller {
			optPGO(c, t)
		}
		if enablePropeller {
			optPropeller(c, t)
		}
		if enablePGOAndPropeller {
			optPGOAndPropeller(c, t)
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
	rootCmd.AddCommand(versionCmd, buildCmd, configCmd, testCmd, optCmd)

	buildCmd.Flags().IntVarP(&jobs, "jobs", "j", 1, "number of jobs")
	buildCmd.Flags().StringVarP(&target, "target", "t", "", "target of the build")
	buildCmd.Flags().StringVarP(&config, "config", "", "", "config of the build")
	buildCmd.Flags().StringVarP(&test_settings, "test-settings", "", "", "the path of test settings")
}
