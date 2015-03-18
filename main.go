package main

import (
	"flag"
	"fmt"
	"github.com/nvlled/gost/genv"
	"github.com/nvlled/gost/util"
	"os"
	"path"
)

var srcDir string
var destDir string
var buildFile string
var showHelp bool
var verbose bool
var verbatimList []string

var defaultOpts = func() *gostOpts {
	emptyStr := ""
	true_ := true
	false_ := false
	return &gostOpts{
		srcDir:   &emptyStr,
		destDir:  &emptyStr,
		optsfile: &emptyStr,
		help:     &false_,
		verbose:  &true_,
	}
}()

func usage(prog string) {
	fmt.Printf("Usage: %s [options] action args...\n", prog)
	println("actions:")
	for name, _ := range actions {
		println("  ", name)
	}
	println("options:")
	flag.PrintDefaults()
}

func main() {
	prog := os.Args[0]

	if len(os.Args) < 2 {
		println("insufficient args")
		return
	}

	fileOpts := new(gostOpts)
	cliOpts := parseArgs(os.Args[1:])
	if cliOpts.optsfile != nil {
		opts, err := readOptsFile(*cliOpts.optsfile)
		if err != nil {
			println("*** failed to read optsfile")
			println(err.Error())
			return
		}
		fileOpts = opts
	}
	opts := defaultOpts.merge(fileOpts).merge(cliOpts)

	println("srcDir: ", *opts.srcDir)
	println("destDir: ", *opts.destDir)
	println("help: ", *opts.help)
	println("verbose: ", *opts.verbose)

	return

	if buildFile != "" {
		err := readBuildFile(buildFile)
		if buildFile != "gostbuild" && err != nil {
			println("Error reading buildfile: ", err.Error())
		}
	}
	if showHelp || (len(os.Args) < 2 && srcDir == "") {
		usage(prog)
		return
	}
	if !validateArgs() {
		return
	}

	srcDir = util.AddTrailingSlash(srcDir)
	destDir = util.AddTrailingSlash(destDir)

	args := flag.Args()
	if len(args) == 0 {
		usage(prog)
		return
	}

	name := args[0]
	args = args[1:]
	action, ok := actions[name]

	if !ok {
		println("unknown action:", name)
	} else {
		action(args)
	}
}

func validateArgs() bool {
	if srcDir == "" {
		fmt.Printf("source directory required\n")
		return false
	}
	if destDir == "" {
		fmt.Printf("destination directory required\n")
		return false
	}
	info, err := os.Lstat(srcDir)
	if err != nil {
		fmt.Printf("failed to open directory: %s\n", srcDir)
		return false
	}
	if !info.IsDir() {
		fmt.Printf("%s is not a directory\n", srcDir)
		return false
	}
	if srcDir == destDir {
		fmt.Printf("source and destination must not be the same\n")
		return false
	}
	return true
}

func printLog(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}

func readBuildFile(filename string) error {
	env, err := genv.ReadFile(filename)
	if err != nil {
		return err
	}
	cwd := path.Dir(buildFile)
	if srcDir == "" {
		srcDir = path.Join(cwd, env.Get("src"))
	}
	if destDir == "" {
		destDir = path.Join(cwd, env.Get("dest"))
	}
	return nil
}
