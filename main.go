package main

import (
	"flag"
	"fmt"
	"github.com/nvlled/gost/genv"
	"github.com/nvlled/gost/util"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// TODO: exclude files from index too

const (
	// recognized env (recenv) keys
	recenvPrefix = ""
	layoutKey    = recenvPrefix + "layout"
	protoKey     = recenvPrefix + "proto"

	relativeKey = recenvPrefix + "relative-url"
	includesKey = recenvPrefix + "includes-dir"
	layoutsKey  = recenvPrefix + "layouts-dir"
	protosKey   = recenvPrefix + "protos-dir"
	verbatimKey = recenvPrefix + "verbatim-files"
	excludesKey = recenvPrefix + "exclude-files"

	protoOpenDelim  = "[["
	protoCloseDelim = "]]"
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
		env:      &emptyStr,
	}
}()

var defaultIncludesDir = "includes"
var defaultLayoutsDir = "layouts"
var defaultProtosDir = "protos"

var defaultVerbatimList = []predicate{}
var defaultExcludesList = []predicate{
	isDotFile,
	baseIs(MARKER_NAME),
	baseIs(genv.FILENAME),
	dirIsVar("includesDir"),
	dirIsVar("layoutsDir"),
	dirIsVar("protosDir"),
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
	baseDir, _ := filepath.Abs(".")

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
		if err != nil {
			baseDir = path.Dir(baseDir)
			optsfile := path.Join(baseDir, *defaultOpts.optsfile)
			opts, err = readOptsFile(optsfile, defaultOpts)
		}
		if err != nil {
			println("*** failed to read optsfile")
			println(err.Error())
			return
		}

		fileOpts = opts
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

	srcDir := util.AddTrailingSlash(*opts.srcDir)
	destDir := util.AddTrailingSlash(*opts.destDir)
	state := newState(srcDir, destDir)

	// envs specified in the command line takes priority over
	// the baseEnv (the env file in the src directory).
	fileEnv := genv.ReadDir(*opts.srcDir)
	env := genv.Parse(strings.Replace(*opts.env, ";", "\n", -1))
	env.SetParent(fileEnv)

	state.baseEnv = env
	state.setIncludesDir(env.GetOr(includesKey, defaultIncludesDir))
	state.setLayoutsDir(env.GetOr(layoutsKey, defaultLayoutsDir))
	state.setProtosDir(env.GetOr(protosKey, defaultProtosDir))

	fn := func(name string) []string {
		paths := strings.Fields(env.Get(name))
		// paths are relative to srcDir
		return paths
	}
	state.setVerbatimList(
		append(
			defaultVerbatimList,
			predicateList(subPathIs, fn(verbatimKey))...,
		),
	)
	state.setExcludeList(
		append(
			defaultExcludesList,
			predicateList(subPathIs, fn(excludesKey))...,
		),
	)
	return state
}
