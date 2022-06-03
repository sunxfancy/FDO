/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import "testing"

func Test_checkToolSets(t *testing.T) {
	type args struct {
		name string
		args []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{"cmake", args{"cmake", []string{"--version"}}, true},
		{"clang", args{"clang", []string{}}, true},
		{"tvb", args{"tvb", []string{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkToolSets(tt.args.name, tt.args.args...); got != tt.want {
				t.Errorf("checkToolSets() = %v, want %v", got, tt.want)
			}
		})
	}
}
