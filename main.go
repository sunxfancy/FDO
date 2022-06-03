/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
	"FDO/cmd"
	"os"
)

func main() {
	if cmd.CheckRequiredToolSets() {
		cmd.Execute()
	} else {
		os.Exit(1)
	}
}
