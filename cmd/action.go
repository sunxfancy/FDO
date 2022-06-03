package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func checkToolSets(name string, args ...string) bool {
	if len(args) == 0 {
		args = []string{"-v"}
	}
	cmd := exec.Command(name, args...)
	if cmd.Run() != nil {
		fmt.Printf("%s is not installed.\n", name)
		return false
	}
	return true
}

func CheckRequiredToolSets() bool {
	return true
	return checkToolSets("cmake", "--version") &&
		checkToolSets("clang") &&
		checkToolSets("lld") &&
		checkToolSets("perf") &&
		checkToolSets("create_llvm_prof")
}

func labelFlags(use_lto bool) []string {
	var k = []string{
		"-funique-internal-linkage-name",
		"-fbasic-block-sections=labels",
	}

	return k
}

func pgoFlags() []string {
	return []string{}
}

// lang = C or CXX
func toCMakeFlags(lang string, flags []string) string {
	return fmt.Sprintf("-DCMAKE_%s_FLAGS=\"%s\"", lang, strings.Join(flags, " "))
}

// kind = EXE or SHARED or MODULE
func toLinkerFlags(kind string, flags []string) string {
	return fmt.Sprintf("-DCMAKE_%s_LINKER_FLAGS=\"%s\"", kind, strings.Join(flags, " "))
}

// This is for PGO
func buildInstrumented(c Config) {

}

// This is for Propeller
func buildLabeled(c Config) {

}

// This is for PGO+Propeller
func buildLabeledOnPGO(c Config) {

}

func testPGO(c Config) {
	testScript := LoadTestScript(c.Source + "/FDO_test.yaml")
	for _, test := range testScript.Commands {
		fmt.Print(test)
	}
}

func testPropeller(c Config) {
	testScript := LoadTestScript(c.Source + "/FDO_test.yaml")
	for _, test := range testScript.Commands {
		fmt.Print(test)
	}
}

func ConfigDir(source string, args []string) {
	p, err := filepath.Abs(source)
	if err != nil {
		fmt.Println("can not get the path: " + source)
	}
	var config = Config{
		Source: p,
		Args:   args,
	}
	StoreConfig("FDO_settings.yaml", config)
}
