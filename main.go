package main

import (
	"flag"
	"fmt"
	"github.com/nvlled/gost/genv"
	"github.com/nvlled/gost/util"
	"os"
)

var srcDir string
var destDir string
var buildFile string
var showHelp bool
var verbose bool
var verbatimList []string

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
	parseArgs()
	if buildFile != "" {
		readBuildFile(buildFile)
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

func parseArgs() {
	flag.StringVar(&srcDir, "src", "", "source files")
	flag.StringVar(&destDir, "dest", "", "destination files")
	flag.StringVar(&buildFile, "build", "gostbuild", "build file")
	flag.BoolVar(&showHelp, "help", false, "show help")
	flag.BoolVar(&verbose, "verbose", true, "show verbose output")
	flag.Parse()
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

func readBuildFile(path string) {
	env := genv.ReadFile(path)
	if srcDir == "" {
		srcDir = env.Get("src")
	}
	if destDir == "" {
		destDir = env.Get("dest")
	}
}
