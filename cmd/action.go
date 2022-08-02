package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Run -v to check a tool is installed
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

type CommandPath struct {
	cmakePath          string
	clangPath          string
	lldPath            string
	perfPath           string
	llvm_profdata      string
	createLlvmProfPath string
	createRegProfPath  string
	dryrun             bool
}

func (c Config) getAbs(p string) string {
	if !filepath.IsAbs(p) {
		test, _ := filepath.Abs(filepath.Dir(c.TestCfg))
		p, _ = filepath.Abs(test + "/" + p)
	}
	return p
}

func (t TestScript) getCommand(c Config) (cmd CommandPath) {
	cmd = CommandPath{"cmake", "clang", "ld.lld", "perf", "llvm-profdata", "create_llvm_prof", "create_reg_prof", c.DryRun}
	if t.ClangPath != "" {
		cmd.clangPath = c.getAbs(t.ClangPath + "/clang")
		cmd.lldPath = c.getAbs(t.ClangPath + "/ld.lld")
		cmd.llvm_profdata = c.getAbs(t.ClangPath + "/llvm-profdata")
	}
	if t.PropellerPath != "" {
		cmd.createLlvmProfPath = c.getAbs(t.PropellerPath + "/create_llvm_prof")
	}
	if t.RegPath != "" {
		cmd.createRegProfPath = c.getAbs(t.RegPath)
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
	default:
		call = cmd
	}
	return call
}

func (c CommandPath) PrintCommand(cmd string, args ...string) {
	fmt.Printf("%s %s\n", c.getPath(cmd), strings.Join(args, " "))
}

func RunWithMultiWriter(command *exec.Cmd) {
	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)

	command.Stdout = mw
	command.Stderr = mw
	command.Run()
}

func (c CommandPath) RunCommand(cmd string, args ...string) {
	c.PrintCommand(cmd, args...)
	if c.dryrun {
		return
	}
	command := exec.Command(c.getPath(cmd), args...)
	RunWithMultiWriter(command)
}

func (c CommandPath) RunShell(cmd string, env ...string) {
	c.PrintCommand("RunShell: " + cmd)
	if c.dryrun {
		return
	}
	s := strings.Split(cmd, " ")
	command := exec.Command(c.getPath(s[0]), s[1:]...)
	command.Env = os.Environ()
	command.Env = append(command.Env, env...)
	RunWithMultiWriter(command)
}

func (c CommandPath) RunCMakeBuild(cfg Config) {
	numOfCores := runtime.NumCPU()
	if cfg.Install {
		c.RunCommand("cmake", "--build", ".", "-j", fmt.Sprint(numOfCores), "--target", "install")
	} else {
		c.RunCommand("cmake", "--build", ".", "-j", fmt.Sprint(numOfCores))
	}
}

func toCMakeCompiler(lang string, path string) string {
	return fmt.Sprintf("-DCMAKE_%s_COMPILER=%s", lang, path)
}

// lang = C or CXX
func toCMakeFlags(lang string, flags ...string) string {
	return fmt.Sprintf("-DCMAKE_%s_FLAGS=%s", lang, strings.Join(flags, " "))
}

// kind = EXE or SHARED or MODULE
func toCMakeLinkerFlags(kind string, flags ...string) string {
	return fmt.Sprintf("-DCMAKE_%s_LINKER_FLAGS=%s", kind, strings.Join(flags, " "))
}

// This function will merge the arguments which has the same key
// e.g. If we have -DCMAKE_C_FLAGS=-O3  -DCMAKE_C_FLAGS=-g will become -DCMAKE_C_FLAGS=-O3 -g
func merge_args(args []string) []string {
	var ans []string

	m := make(map[string]string)
	for _, v := range args {
		s := strings.SplitN(v, "=", 2)
		if len(s) != 2 {
			ans = append(ans, v)
			continue
		}
		_, prs := m[s[0]]
		if !prs {
			m[s[0]] = s[1]
		} else {
			m[s[0]] = m[s[0]] + " " + s[1]
		}
	}
	for k, v := range m {
		ans = append(ans, k+"="+v)
	}
	return ans
}

type CMakeFlags struct {
	Config
	flags        []string
	linker_flags []string
	install_path string
}

func (f CMakeFlags) createCMakeArgs(cmd CommandPath, t TestScript, s ...string) []string {
	var args = []string{
		f.Source,
		toCMakeCompiler("C", cmd.getPath("clang")), toCMakeCompiler("CXX", cmd.getPath("clang++")),
		toCMakeFlags("C", f.flags...), toCMakeFlags("CXX", f.flags...),
		toCMakeLinkerFlags("EXE", f.linker_flags...), toCMakeLinkerFlags("SHARED", f.linker_flags...), toCMakeLinkerFlags("MODULE", f.linker_flags...),
	}
	if f.install_path != "" {
		args = append(args, "-DCMAKE_INSTALL_PREFIX="+f.install_path)
	}

	args = append(args, f.Args...)
	args = append(args, s...)
	return merge_args(args)
}

func createDefaultFlags(c Config) CMakeFlags {
	return CMakeFlags{c, []string{"-fuse-ld=lld"}, []string{"-fuse-ld=lld"}, ""}
}

func (f CMakeFlags) PGO(stage string) CMakeFlags {
	if stage == "instrumented" {
		f.flags = append(f.flags, fmt.Sprint("-f", f.Profile, "-generate"))
		if f.Install {
			path, _ := os.Getwd()
			f.install_path = path + "/instrumented/install"
		}
	}
	if stage == "pgo-opt" {
		profdata_path := ""
		if f.Install {
			profdata_path, _ = filepath.Abs("../instrumented/install/PGO.profdata")
		} else {
			profdata_path, _ = filepath.Abs("../instrumented/PGO.profdata")
		}
		f.flags = append(f.flags, fmt.Sprint("-f", f.Profile, "-use="+profdata_path))
	}

	return f
}

func (f CMakeFlags) IPRA() CMakeFlags {
	if f.lto == "" && f.ipra { // disable LTO
		f.flags = append(f.flags, "-enable-ipra")
	}
	if f.lto != "" && f.ipra { // enable LTO
		f.linker_flags = append(f.linker_flags, "-Wl,-mllvm -Wl,-enable-ipra")
	}
	return f
}

func (f CMakeFlags) LTO() CMakeFlags {
	if f.lto != "" {
		f.flags = append(f.flags, "-flto="+f.lto)
		f.linker_flags = append(f.linker_flags, "-flto="+f.lto)
	}
	return f
}

func (f CMakeFlags) Propeller(stage string) CMakeFlags {
	if stage == "labeled" || stage == "labeled-opt" {
		f.flags = append(f.flags, "-funique-internal-linkage-names", "-fbasic-block-sections=labels")
		if f.lto != "" { // enable LTO
			f.linker_flags = append(f.linker_flags, "-Wl,--lto-basic-block-sections=labels")
		}
		if f.Install {
			path, _ := os.Getwd()
			f.install_path = path + "/" + stage + "/install"
		}
	}
	if stage == "propeller-opt" || stage == "final-opt" {
		profdata_path := ""
		labeled := "labeled"
		if stage == "final-opt" {
			labeled += "-pgo"
		}
		if f.Install {
			profdata_path, _ = filepath.Abs("../" + labeled + "/install")
		} else {
			profdata_path, _ = filepath.Abs("../" + labeled)
		}
		symorder := profdata_path + "/symorder.txt"
		cluster := profdata_path + "/cluster.txt"

		f.flags = append(f.flags, "-funique-internal-linkage-names", "-fbasic-block-sections=list="+cluster)
		f.linker_flags = append(f.linker_flags, "-Wl,--no-warn-symbol-ordering", "-Wl,--symbol-ordering-file="+symorder)

		if f.lto != "" {
			f.linker_flags = append(f.linker_flags, "-Wl,--lto-basic-block-sections="+cluster)
		}
	}
	return f
}

func createAndMoveToFolder(name string) {
	err := os.MkdirAll(name, 0777)
	if err != nil {
		fmt.Println("mkdir faild: " + name)
		panic(err)
	}
	path, err := filepath.Abs("./" + name)
	if os.Chdir(path) != nil {
		fmt.Println("can not change to the path: " + path)
		panic(err)
	}
}

// This is for PGO
func buildInstrumented(c Config, t TestScript) {
	cmd := t.getCommand(c)
	createAndMoveToFolder("instrumented")
	flags := createDefaultFlags(c).PGO("instrumented").IPRA().LTO()
	var args = flags.createCMakeArgs(cmd, t)
	cmd.RunCommand("cmake", args...)
	cmd.RunCMakeBuild(c)
	os.Chdir("..")
}

// This is for Propeller
func buildLabeled(c Config, t TestScript) {
	cmd := t.getCommand(c)
	createAndMoveToFolder("labeled")
	flags := createDefaultFlags(c).Propeller("labeled").IPRA().LTO()
	var args = flags.createCMakeArgs(cmd, t)
	cmd.RunCommand("cmake", args...)
	cmd.RunCMakeBuild(c)
	os.Chdir("..")
}

// This is for PGO+Propeller
func buildLabeledOnPGO(c Config, t TestScript) {
	cmd := t.getCommand(c)
	createAndMoveToFolder("labeled-pgo")

	flags := createDefaultFlags(c).PGO("pgo-opt").Propeller("labeled-opt").IPRA().LTO()
	var args = flags.createCMakeArgs(cmd, t)

	cmd.RunCommand("cmake", args...)
	cmd.RunCMakeBuild(c)
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
	cmd := t.getCommand(c)
	moveToTestFolder(c, "instrumented")
	// First, run those tests
	for k, test := range t.Commands {
		cmd.RunShell(test, fmt.Sprint("LLVM_PROFILE_FILE=PGO", k, ".profraw"))
	}
	// Then, combine the profiles
	var files = searchProfraw()
	var nargs = []string{"merge", "-output=PGO.profdata"}
	nargs = append(nargs, files...)
	cmd.RunCommand("llvm-profdata", nargs...)
	moveBack(c)
}

func testPropeller(c Config, t TestScript) {
	cmd := t.getCommand(c)
	moveToTestFolder(c, "labeled")
	for k, test := range t.Commands {
		cmd.RunShell(fmt.Sprint("perf record -e cycles:u -j any,u -o Propeller", k, ".data -- ", test))
	}
	binary_path, _ := filepath.Abs(t.Binary)
	// TODO: here we need to handle multiple profiles
	cmd.RunCommand("create_llvm_prof", "--format=propeller", "--binary="+binary_path,
		"--profile=Propeller0.data", "--out=cluster.txt", "--propeller_symorder=symorder.txt")
	moveBack(c)
}

func testPropellerOnPGO(c Config, t TestScript) {
	cmd := t.getCommand(c)
	moveToTestFolder(c, "labeled-pgo")
	for k, test := range t.Commands {
		cmd.RunShell(fmt.Sprint("perf record -e cycles:u -j any,u -o Propeller", k, ".data -- ", test))
	}
	binary_path, _ := filepath.Abs(t.Binary)
	// TODO: here we need to handle multiple profiles
	cmd.RunCommand("create_llvm_prof", "--format=propeller", "--binary="+binary_path,
		"--profile=Propeller0.data", "--out=cluster.txt", "--propeller_symorder=symorder.txt")
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

// build the optimized binary using PGO.profdata
func optPGO(c Config, t TestScript, args []string) {
	cmd := t.getCommand(c)
	createAndMoveToFolder("pgo-opt")

	flags := createDefaultFlags(c).PGO("pgo-opt").IPRA().LTO()
	var cargs = flags.createCMakeArgs(cmd, t, args...)

	cmd.RunCommand("cmake", cargs...)
	cmd.RunCMakeBuild(c)
	os.Chdir("..")
}

func optPropeller(c Config, t TestScript, args []string) {
	// First, convert the profile data
	cmd := t.getCommand(c)
	createAndMoveToFolder("labeled-opt")
	flags := createDefaultFlags(c).Propeller("propeller-opt").IPRA().LTO()
	var cargs = flags.createCMakeArgs(cmd, t, args...)

	cmd.RunCommand("cmake", cargs...)
	cmd.RunCMakeBuild(c)
	os.Chdir("..")
}

func optPGOAndPropeller(c Config, t TestScript, args []string) {
	// First, convert the profile data
	cmd := t.getCommand(c)
	createAndMoveToFolder("final-pgo")
	flags := createDefaultFlags(c).PGO("pgo-opt").Propeller("final-opt").IPRA().LTO()
	var cargs = flags.createCMakeArgs(cmd, t, args...)

	cmd.RunCommand("cmake", cargs...)
	cmd.RunCMakeBuild(c)
	os.Chdir("..")
}

func ConfigDir(source string, lto string, args []string) {
	p, err := filepath.Abs(source)
	if err != nil {
		fmt.Println("can not get the path: " + source)
		panic(err)
	}
	var config = Config{
		Source: p,
		Args:   args,
		lto:    lto,
	}
	config.StoreConfig("FDO_settings.yaml")
}
