package cmd

import (
	"bytes"
	"fmt"
	"io"
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

// t = full or thin
func ltoFlags(t string) []string {
	if t != "" {
		t = "=" + t
	}
	return []string{
		"-flto" + t,
	}
}

func labelFlags(use_lto string) []string {
	var k = []string{
		"-funique-internal-linkage-names",
		"-fbasic-block-sections=labels",
	}
	if use_lto != "" {
		k = append(k, ltoFlags(use_lto)...)
	}
	return k
}

func labelUseFlags(use_lto string) []string {
	cluster, _ := filepath.Abs("../labeled/cluster.txt")

	var k = []string{
		"-funique-internal-linkage-names",
		"-fbasic-block-sections=list=" + cluster,
	}
	if use_lto != "" {
		k = append(k, ltoFlags(use_lto)...)
	}
	return k
}

func toCMakeLinkerFlags(kind string, flags ...string) string {
	return fmt.Sprintf("-DCMAKE_%s_LINKER_FLAGS=%s", kind, strings.Join(flags, " "))
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

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)

	command.Stdout = mw
	command.Stderr = mw
	command.Run()
}

func (c CommandPath) RunShell(cmd string, env ...string) {
	fmt.Println("RunShell: " + cmd)
	s := strings.Split(cmd, " ")
	command := exec.Command(s[0], s[1:]...)
	command.Env = os.Environ()
	command.Env = append(command.Env, env...)

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)

	command.Stdout = mw
	command.Stderr = mw
	command.Run()
}

func createCMakeArgs(c Config, t TestScript, flags []string, linker_flags []string) []string {
	cmd := t.getCommand()
	var args = []string{
		c.Source,
		toCMakeCompiler("C", cmd.getPath("clang")), toCMakeCompiler("CXX", cmd.getPath("clang++")),
		toCMakeFlags("C", flags...), toCMakeFlags("CXX", flags...),
		toCMakeLinkerFlags("EXE", linker_flags...), toCMakeLinkerFlags("SHARED", linker_flags...), toCMakeLinkerFlags("MODULE", linker_flags...),
	}
	if c.Install {
		path, _ := os.Getwd()
		args = append(args, "-DCMAKE_INSTALL_PREFIX="+path+"/install")
	}
	args = append(args, c.Args...)
	return args
}

func (cmd CommandPath) runCMakeBuild(c Config) {
	if c.Install {
		cmd.RunCommand("cmake", "--build", ".", "-j", "8", "--target", "install")
	} else {
		cmd.RunCommand("cmake", "--build", ".", "-j", "8")
	}
}

// This is for PGO
func buildInstrumented(c Config, t TestScript) {
	cmd := t.getCommand()
	os.MkdirAll("instrumented", 0777)
	path, _ := filepath.Abs("./instrumented")
	if os.Chdir(path) != nil {
		fmt.Println("can not change to the path: " + path)
	}
	instrument_flags := []string{"-fprofile-instr-generate"}
	linker_flags := []string{"-fuse-ld=lld"}

	var args = createCMakeArgs(c, t, instrument_flags, linker_flags)
	cmd.RunCommand("cmake", args...)
	cmd.runCMakeBuild(c)
	os.Chdir("..")
}

// This is for Propeller
func buildLabeled(c Config, t TestScript) {
	cmd := t.getCommand()
	os.MkdirAll("labeled", 0777)
	os.Chdir("labeled")

	linker_flags := []string{"-fuse-ld=lld"}
	if c.LTO != "" {
		linker_flags = append(linker_flags, "-Wl,--lto-basic-block-sections=labels")
	}
	var args = createCMakeArgs(c, t, labelFlags(c.LTO), linker_flags)
	cmd.RunCommand("cmake", args...)
	cmd.runCMakeBuild(c)

	os.Chdir("..")
}

// This is for PGO+Propeller
func buildLabeledOnPGO(c Config, t TestScript) {
	cmd := t.getCommand()
	os.MkdirAll("labeled-pgo", 0777)
	os.Chdir("labeled-pgo")

	profdata_path, _ := filepath.Abs("../instrumented/PGO.profdata")
	flags := []string{"-fprofile-instr-use=" + profdata_path}
	flags = append(flags, labelFlags(c.LTO)...)
	linker_flags := []string{"-fuse-ld=lld"}
	if c.LTO != "" {
		linker_flags = append(linker_flags, "-Wl,--lto-basic-block-sections=labels")
	}
	var args = createCMakeArgs(c, t, flags, linker_flags)
	cmd.RunCommand("cmake", args...)
	cmd.runCMakeBuild(c)

	os.Chdir("..")
}

func moveToTestFolder(c Config, name string) {
	os.Chdir(name)
	if c.Install {
		os.Chdir("install")
	}
}

func moveBack(c Config) {
	os.Chdir("..")
	if c.Install {
		os.Chdir("..")
	}
}

func testPGO(c Config, t TestScript) {
	cmd := t.getCommand()
	moveToTestFolder(c, "instrumented")
	for k, test := range t.Commands {
		cmd.RunShell(test, "LLVM_PROFILE_FILE=PGO"+fmt.Sprint(k)+".profraw")
	}
	moveBack(c)
}

func testPropeller(c Config, t TestScript) {
	cmd := t.getCommand()
	moveToTestFolder(c, "labeled")
	for k, test := range t.Commands {
		cmd.RunShell("perf record -e cycles:u -j any,u -o Propeller" + fmt.Sprint(k) + ".data -- " + test)
	}
	moveBack(c)
}

func testPGOAndPropeller(c Config, t TestScript) {
	cmd := t.getCommand()
	moveToTestFolder(c, "labeled-pgo")
	for k, test := range t.Commands {
		cmd.RunShell("perf record -e cycles:u -j any,u -o Propeller" + fmt.Sprint(k) + ".data -- " + test)
	}
	moveBack(c)
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
	flags := []string{"-fprofile-instr-use=" + profdata_path}
	linker_flags := []string{"-fuse-ld=lld"}
	var args = createCMakeArgs(c, t, flags, linker_flags)

	args = append(args, c.Args...)
	cmd.RunCommand("cmake", args...)
	cmd.runCMakeBuild(c)

	os.Chdir("..")
}

func optPropeller(c Config, t TestScript) {
	// First, convert the profile data
	cmd := t.getCommand()
	os.Chdir("labeled")
	binary_path, _ := filepath.Abs(t.Binary)
	// TODO: here we need to handle multiple profiles
	cmd.RunCommand("create_llvm_prof", "--format=propeller", "--binary="+binary_path,
		"--profile=Propeller0.data", "--out=cluster.txt", "--propeller_symorder=symorder.txt")
	os.Chdir("..")

	os.MkdirAll("labeled-opt", 0777)
	os.Chdir("labeled-opt")
	symorder, _ := filepath.Abs("../labeled/symorder.txt")
	linker_flags := []string{"-fuse-ld=lld", "-Wl,--no-warn-symbol-ordering", "-Wl,--symbol-ordering-file=" + symorder}
	if c.LTO != "" {
		cluster, _ := filepath.Abs("../labeled/cluster.txt")
		linker_flags = append(linker_flags, "-Wl,--lto-basic-block-sections="+cluster)
	}
	var args = createCMakeArgs(c, t, labelUseFlags(c.LTO), linker_flags)

	cmd.RunCommand("cmake", args...)
	cmd.runCMakeBuild(c)

	os.Chdir("..")
}

func optPGOAndPropeller(c Config, t TestScript) {
	// First, convert the profile data
	cmd := t.getCommand()
	os.Chdir("labeled-pgo")
	binary_path, _ := filepath.Abs(t.Binary)
	// TODO: here we need to handle multiple profiles
	cmd.RunCommand("create_llvm_prof", "--format=propeller", "--binary="+binary_path,
		"--profile=Propeller0.data", "--out=cluster.txt", "--propeller_symorder=symorder.txt")
	os.Chdir("..")

	os.MkdirAll("final-opt", 0777)
	os.Chdir("final-opt")
	profdata_path, _ := filepath.Abs("../instrumented/PGO.profdata")
	flags := []string{"-fprofile-instr-use=" + profdata_path}
	flags = append(flags, labelFlags(c.LTO)...)
	symorder, _ := filepath.Abs("../labeled-pgo/symorder.txt")
	linker_flags := []string{"-fuse-ld=lld", "-Wl,--no-warn-symbol-ordering", "-Wl,--symbol-ordering-file=" + symorder}
	if c.LTO != "" {
		cluster, _ := filepath.Abs("../labeled-pgo/cluster.txt")
		linker_flags = append(linker_flags, "-Wl,--lto-basic-block-sections="+cluster)
	}
	var args = createCMakeArgs(c, t, labelUseFlags(c.LTO), linker_flags)

	cmd.RunCommand("cmake", args...)
	cmd.runCMakeBuild(c)

	os.Chdir("..")
}

func ConfigDir(source string, lto string, args []string) {
	p, err := filepath.Abs(source)
	if err != nil {
		fmt.Println("can not get the path: " + source)
	}
	var config = Config{
		Source: p,
		Args:   args,
		LTO:    lto,
	}
	config.StoreConfig("FDO_settings.yaml")
}
