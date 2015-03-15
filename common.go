package main

import (
	"fmt"
	"github.com/nvlled/gost/defaults"
	"github.com/nvlled/gost/genv"
	fpath "path/filepath"
	"strings"
)

const (
	// distdel: directory is safe to delete
	MARKER_NAME       = ".gost-distdel"
	VERBATIM_KEY      = "verbatim"
	INCLUDES_DIR_KEY  = "includes-dir"
	LAYOUTS_DIR_KEY   = "layouts-dir"
	TEMPLATES_DIR_KEY = "templates-dir"
)

type Index map[string]genv.T

var index Index
var pathIndex Index

var includesDir string
var layoutsDir string
var templatesDir string

var baseEnv = genv.T{
	INCLUDES_DIR_KEY:  defaults.INCLUDES_DIR,
	LAYOUTS_DIR_KEY:   defaults.LAYOUTS_DIR,
	TEMPLATES_DIR_KEY: defaults.TEMPLATES_DIR,
}

func printLog(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}

func isVerbatim(env genv.T, path string) bool {
	if env != nil {
		for _, pref := range verbatimList {
			if strings.HasPrefix(path, pref) {
				return true
			}
		}
	}
	return false
}

func errHandler() {
	err := recover()
	if err != nil {
		fmt.Printf("*** error %v\n", err)
	}
}

func isItemplate(path string) bool {
	ext := fpath.Ext(path)
	return ext == ".html" ||
		ext == ".js" ||
		ext == ".css"

}

func skipFile(file string) bool {
	base := fpath.Base(file)
	dir := fpath.Dir(file)
	return strings.HasPrefix(base, ".") || //exclude dot files
		base == genv.FILENAME ||
		dir == includesDir ||
		dir == layoutsDir ||
		dir == templatesDir ||
		base == MARKER_NAME
}
