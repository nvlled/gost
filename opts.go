package main

import (
	"flag"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
)

type gostOpts struct {
	// use pointers to allow nullability
	srcDir   *string
	destDir  *string
	optsfile *string
	help     *bool
	verbose  *bool
}

// * merges opts and opts_
// * non-nil values in opts_ takes priority
// * no mutation is done in both given opts
func (opts *gostOpts) merge(opts_ *gostOpts) *gostOpts {
	newOpts := *opts
	if opts_.srcDir != nil {
		newOpts.srcDir = opts_.srcDir
	}
	if opts_.destDir != nil {
		newOpts.destDir = opts_.destDir
	}
	if opts_.optsfile != nil {
		newOpts.optsfile = opts_.optsfile
	}
	if opts_.help != nil {
		newOpts.help = opts_.help
	}
	if opts_.verbose != nil {
		newOpts.verbose = opts_.verbose
	}
	return &newOpts
}

func parseArgs(args []string) (*gostOpts, []string) {
	flagSet := flag.NewFlagSet("flags", flag.ExitOnError)

	// *** Note: default values are ignored ***
	srcDir := flagSet.String("srcDir", "", "source files")
	destDir := flagSet.String("destDir", "", "destination files")
	optsfile := flagSet.String("optsfile", "", "build file")
	help := flagSet.Bool("help", false, "show help")
	verbose := flagSet.Bool("verbose", false, "show verbose output")

	flagSet.Parse(args)

	// opts will have nil values for flags that are not set
	opts := &gostOpts{}
	flagSet.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "srcDir":
			opts.srcDir = srcDir
		case "destDir":
			opts.destDir = destDir
		case "optsfile":
			opts.optsfile = optsfile
		case "help":
			opts.help = help
		case "verbose":
			opts.verbose = verbose
		}
	})
	return opts, flagSet.Args()
}

func readOptsFile(filename string) (*gostOpts, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	args := strings.FieldsFunc(string(bytes), unicode.IsSpace)
	opts, _ := parseArgs(args)
	return opts, nil
}
