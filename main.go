package main

import (
	"flag"
	"fmt"
	"github.com/nvlled/gost/genv"
	"github.com/nvlled/gost/util"
	"os"
	"path"
	"strings"
)

var verbose bool

// do not mutate directly: *defaultOpts.srcDir = "x"
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

var defaultIncludesDir = "includes"
var defaultLayoutsDir = "layouts"
var defaultTemplatesDir = "templates"

var defaultVerbatimList = []predicate{}
var defaultExcludesList = []predicate{
	isDotFile,
	baseIs(MARKER_NAME),
	baseIs(genv.FILENAME),
	dirIsVar("includesDir"),
	dirIsVar("layoutsDir"),
	dirIsVar("templatesDir"),
}

func usage(prog string) {
	fmt.Printf("Usage: %s [options] action args...\n", prog)
	println("actions:")
	for name, _ := range actions {
		println("  ", name)
	}
	println("options:")
	flag.PrintDefaults()
}

func printLog(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}

func main() {
	prog := os.Args[0]

	fileOpts := new(gostOpts)
	cliOpts, args := parseArgs(os.Args[1:])

	// srcDir and destDir are relative to dir of optsfile
	baseDir := ""

	if cliOpts.optsfile != nil {
		baseDir = path.Dir(*cliOpts.optsfile)
		opts, err := readOptsFile(*cliOpts.optsfile)
		if err != nil {
			println("*** failed to read optsfile")
			println(err.Error())
			return
		}
		fileOpts = opts
	}
	opts := defaultOpts.merge(fileOpts).merge(cliOpts)

	prependBase := func(dir string) *string {
		var s string
		s = util.PrependPath(dir, baseDir)
		return &s
	}
	opts.srcDir = prependBase(*opts.srcDir)
	opts.destDir = prependBase(*opts.destDir)

	verbose = *opts.verbose
	if *opts.help || (len(os.Args) < 2 && *opts.srcDir == "") {
		usage(prog)
		return
	}
	if !validateOpts(opts) {
		return
	}
	state := optsToState(opts)

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
		// pikachu elf is fake
		action(state, args)
	}
}

func optsToState(opts *gostOpts) *gostState {
	env := genv.ReadDir(*opts.srcDir)

	srcDir := util.AddTrailingSlash(*opts.srcDir)
	destDir := util.AddTrailingSlash(*opts.destDir)
	state := newState(srcDir, destDir)

	state.setIncludesDir(env.GetOr("includes", defaultIncludesDir))
	state.setLayoutsDir(env.GetOr("layouts", defaultLayoutsDir))
	state.setTemplatesDir(env.GetOr("templates", defaultTemplatesDir))

	fn := func(name string) []string {
		paths := strings.Fields(env.Get(name))
		for i := range paths {
			paths[i] = util.PrependPath(paths[i], srcDir)
		}
		return paths
	}
	state.setVerbatimList(
		append(
			defaultVerbatimList,
			predicateList(pathIs, fn("verbatim"))...,
		),
	)
	state.setExcludeList(
		append(
			defaultExcludesList,
			predicateList(pathIs, fn("excludes"))...,
		),
	)
	return state
}

func validateOpts(opts *gostOpts) bool {
	srcDir := *opts.srcDir
	destDir := *opts.destDir

	// Fix: both srcDir and destDir is set to "." by default
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
