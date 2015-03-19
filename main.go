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
var defaultOptsfile = "gostopts"

// do not mutate directly: *defaultOpts.srcDir = "x"
var defaultOpts = func() *gostOpts {
	emptyStr := ""
	true_ := true
	false_ := false
	return &gostOpts{
		srcDir:   &emptyStr,
		destDir:  &emptyStr,
		optsfile: &defaultOptsfile,
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

var itemplates = []string{".html", ".js", ".css"}

func usage(prog string, flagSet *flag.FlagSet) {
	indent := "  "
	fmt.Printf("Usage: %s [options] action args...\n", prog)
	println("actions:")
	for name, _ := range actions {
		println(indent, name)
	}
	println("options:")
	flagSet.PrintDefaults()
	println("action help:")
	println(indent, "Specify both -help and the action to show help for each action")
	fmt.Printf("%s%s --help <action>\n", indent, prog)
}

func printLog(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}

func main() {
	prog := os.Args[0]

	fileOpts := new(gostOpts)
	cliOpts, flagSet := parseArgs(os.Args[1:], defaultOpts)

	// srcDir and destDir are relative to dir of optsfile
	baseDir := "."

	if cliOpts.optsfile != nil {
		baseDir = path.Dir(*cliOpts.optsfile)
		opts, err := readOptsFile(*cliOpts.optsfile, defaultOpts)
		if err != nil {
			println("*** failed to read optsfile")
			println(err.Error())
			return
		}
		fileOpts = opts
	} else {
		opts, err := readOptsFile(*defaultOpts.optsfile, defaultOpts)
		if err == nil {
			fileOpts = opts
		}
	}
	opts := defaultOpts.merge(fileOpts).merge(cliOpts)

	prependBase := func(dir string) *string {
		if dir == "" {
			// avoid returning "." when dir is ""
			return &dir
		}
		var s string
		s = util.PrependPath(dir, baseDir)
		return &s
	}
	opts.srcDir = prependBase(*opts.srcDir)
	opts.destDir = prependBase(*opts.destDir)

	verbose = *opts.verbose
	if len(os.Args) < 2 && *opts.srcDir == "" {
		usage(prog, flagSet)
		return
	}

	args := flagSet.Args()
	if len(args) == 0 {
		usage(prog, flagSet)
		return
	}

	name := args[0]
	action, ok := actions[name]

	if !ok {
		println("unknown action:", name)
	} else if *opts.help {
		fmt.Printf(action.help, prog, name)
		println()
	} else {
		defer handleValidation()
		action.fn(opts, args)
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
		// paths are relative to srcDir
		return paths
	}
	state.setVerbatimList(
		append(
			defaultVerbatimList,
			predicateList(subPathIs, fn("verbatim"))...,
		),
	)
	state.setExcludeList(
		append(
			defaultExcludesList,
			predicateList(subPathIs, fn("excludes"))...,
		),
	)
	return state
}
