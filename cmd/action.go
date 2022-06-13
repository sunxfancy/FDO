package cmd

import (
	"fmt"
	"os"
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

func labelFlags(use_lto bool) []string {
	var k = []string{
		"-funique-internal-linkage-names",
		"-fbasic-block-sections=labels",
	}

	return k
}

func labelUseFlags(use_lto bool) []string {
	var k = []string{
		"-funique-internal-linkage-names",
		"-fbasic-block-sections=labels",
	}

	return k
}

func pgoFlags() []string {
	return []string{}
}

func toCMakeCompiler(lang string, path string) string {
	return fmt.Sprintf("-DCMAKE_%s_COMPILER=%s", lang, path)
}

// lang = C or CXX
func toCMakeFlags(lang string, flags ...string) string {
	return fmt.Sprintf("-DCMAKE_%s_FLAGS=%s", lang, strings.Join(flags, " "))
}

// kind = EXE or SHARED or MODULE
func toLinkerFlags(kind string, flags ...string) string {
	return fmt.Sprintf("-DCMAKE_%s_LINKER_FLAGS=\"%s\"", kind, strings.Join(flags, " "))
}

type CommandPath struct {
	cmakePath          string
	clangPath          string
	lldPath            string
	perfPath           string
	llvm_profdata      string
	createLlvmProfPath string
	createRegProfPath  string
}

func (t TestScript) getCommand() (cmd CommandPath) {
	cmd = CommandPath{"cmake", "clang", "ld.lld", "perf", "llvm-profdata", "create_llvm_prof", "create_reg_prof"}
	if t.ClangPath != "" {
		cmd.clangPath = t.ClangPath + "/clang"
		cmd.lldPath = t.ClangPath + "/ld.lld"
		cmd.llvm_profdata = t.ClangPath + "/llvm-profdata"
	}
	if t.PropellerPath != "" {
		cmd.createLlvmProfPath = t.PropellerPath + "/create_llvm_prof"
	}
	if t.RegPath != "" {
		cmd.createRegProfPath = t.RegPath
	}

	var succes = checkToolSets(cmd.cmakePath, "--version") &&
		checkToolSets(cmd.clangPath) &&
		checkToolSets(cmd.lldPath, "--version") &&
		checkToolSets(cmd.perfPath) &&
		checkToolSets(cmd.createLlvmProfPath, "--version")
	if !succes {
		os.Exit(1)
	}
	return
}

func (c CommandPath) getPath(cmd string) string {
	var call string
	switch cmd {
	case "cmake":
		call = c.cmakePath
	case "clang":
		call = c.clangPath
	case "clang++":
		call = c.clangPath + "++"
	case "lld":
		call = c.lldPath
	case "perf":
		call = c.perfPath
	case "llvm-profdata":
		call = c.llvm_profdata
	case "create_llvm_prof":
		call = c.createLlvmProfPath
	case "create_reg_prof":
		call = c.createRegProfPath
	}
	return call
}

func (c CommandPath) PrintCommand(cmd string, args ...string) {
	fmt.Printf("%s %s\n", c.getPath(cmd), strings.Join(args, " "))
}

func (c CommandPath) RunCommand(cmd string, args ...string) {
	c.PrintCommand(cmd, args...)
	command := exec.Command(c.getPath(cmd), args...)
	stdout, _ := command.CombinedOutput()
	fmt.Println(string(stdout))
}

func (c CommandPath) RunShell(cmd string, env ...string) {
	fmt.Println("RunShell: " + cmd)
	s := strings.Split(cmd, " ")
	command := exec.Command(s[0], s[1:]...)
	command.Env = os.Environ()
	command.Env = append(command.Env, env...)
	stdout, err := command.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(stdout))
}

// This is for PGO
func buildInstrumented(c Config, t TestScript) {
	cmd := t.getCommand()
	os.MkdirAll("instrumented", 0777)
	path, _ := filepath.Abs("./instrumented")
	if os.Chdir(path) != nil {
		fmt.Println("can not change to the path: " + path)
	}
	var args = []string{
		c.Source,
		toCMakeCompiler("C", cmd.getPath("clang")), toCMakeCompiler("CXX", cmd.getPath("clang++")),
		toCMakeFlags("C", "-fprofile-instr-generate"), toCMakeFlags("CXX", "-fprofile-instr-generate"),
	}
	args = append(args, c.Args...)
	cmd.RunCommand("cmake", args...)
	cmd.RunCommand("cmake", "--build", ".")
	os.Chdir("..")
}

// This is for Propeller
func buildLabeled(c Config, t TestScript) {
	cmd := t.getCommand()
	os.MkdirAll("labeled", 0777)
	os.Chdir("labeled")
	var args = []string{
		c.Source,
		toCMakeCompiler("C", cmd.getPath("clang")), toCMakeCompiler("CXX", cmd.getPath("clang++")),
		toCMakeFlags("C", labelFlags(false)...), toCMakeFlags("CXX", labelFlags(false)...),
	}
	cmd.RunCommand("cmake", args...)
	cmd.RunCommand("cmake", "--build", ".")
	os.Chdir("..")
}

// This is for PGO+Propeller
func buildLabeledOnPGO(c Config, t TestScript) {

}

func testPGO(c Config, t TestScript) {
	cmd := t.getCommand()
	os.Chdir("instrumented")
	for k, test := range t.Commands {
		cmd.RunShell(test, "LLVM_PROFILE_FILE=PGO"+fmt.Sprint(k)+".profraw")
	}
	os.Chdir("..")
}

func testPropeller(c Config, t TestScript) {
	cmd := t.getCommand()
	os.Chdir("labeled")
	for k, test := range t.Commands {
		cmd.RunShell("perf record -e cycles:u -j any,u -o Propeller" + fmt.Sprint(k) + ".data -- " + test)
	}
	os.Chdir("..")
}

func testPGOAndPropeller(c Config, t TestScript) {
	cmd := t.getCommand()
	os.Chdir("pgo-labeled")
	for _, test := range t.Commands {
		cmd.RunShell(test)
	}
	os.Chdir("..")
}

func searchProfraw() []string {
	var files []string
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".profraw") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func optPGO(c Config, t TestScript) {
	// First, combine the profiles
	cmd := t.getCommand()
	os.Chdir("instrumented")
	var files = searchProfraw()
	var nargs = []string{"merge", "-output=PGO.profdata"}
	nargs = append(nargs, files...)
	cmd.RunCommand("llvm-profdata", nargs...)
	os.Chdir("..")

	// Then, build the optimized binary using PGO.profdata
	os.MkdirAll("instrumented-opt", 0777)
	path, _ := filepath.Abs("./instrumented-opt")
	if os.Chdir(path) != nil {
		fmt.Println("can not change to the path: " + path)
	}

	profdata_path, _ := filepath.Abs("../instrumented/PGO.profdata")
	var args = []string{
		c.Source,
		toCMakeCompiler("C", cmd.getPath("clang")), toCMakeCompiler("CXX", cmd.getPath("clang++")),
		toCMakeFlags("C", "-fprofile-instr-use="+profdata_path),
		toCMakeFlags("CXX", "-fprofile-instr-use="+profdata_path),
	}
	args = append(args, c.Args...)
	cmd.RunCommand("cmake", args...)
	cmd.RunCommand("cmake", "--build", ".")
	os.Chdir("..")
}

func optPropeller(c Config, t TestScript) {
	// First, convert the profile data
	cmd := t.getCommand()
	os.Chdir("labeled")
	// TODO: here we need to handle multiple profiles
	cmd.RunCommand("create_llvm_prof", "--format=propeller", "--binary="+t.Binary,
		"--profile=Propeller0.data", "--out=cluster.txt", "--propeller_symorder=symorder.txt")
	os.Chdir("..")

	os.MkdirAll("labeled-opt", 0777)
	os.Chdir("labeled-opt")
	var args = []string{
		c.Source,
		toCMakeCompiler("C", cmd.getPath("clang")), toCMakeCompiler("CXX", cmd.getPath("clang++")),
		toCMakeFlags("C", labelFlags(false)...), toCMakeFlags("CXX", labelFlags(false)...),
	}
	cmd.RunCommand("cmake", args...)
	cmd.RunCommand("cmake", "--build", ".")
	os.Chdir("..")
}

func optPGOAndPropeller(c Config, t TestScript) {

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
