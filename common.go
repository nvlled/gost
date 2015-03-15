package main

import (
	"fmt"
	"github.com/nvlled/gost/defaults"
	"github.com/nvlled/gost/genv"
	"github.com/nvlled/gost/util"
	fpath "path/filepath"
	"regexp"
	"strings"
	"text/template"
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

var globalFuncMap = template.FuncMap{
	"genid": util.GenerateId,
	"shell": util.Exec,
}

	}
}

func printLog(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}

func createTemplate() *template.Template {
	funcMap := createFuncMap(".")
	for k, v := range globalFuncMap {
		funcMap[k] = v
	}

	return template.New("default").Funcs(funcMap)
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

func relativizePath(srcPath, destPath string) string {
	re := regexp.MustCompile(`^/`)
	if srcPath == "/" {
		if destPath == "/" {
			return "."
		}
		return strings.TrimPrefix(destPath, "/")
	}
	if !re.MatchString(destPath) {
		return destPath
	}
	if !re.MatchString(srcPath) {
		srcPath = fpath.Join("/", srcPath)
	}

	sep := string(fpath.Separator)
	prefix := util.CommonSubPath(destPath, srcPath) + sep

	srcPath_ := strings.TrimPrefix(srcPath, prefix)
	destPath_ := strings.TrimPrefix(destPath, prefix)

	slevel := util.DirLevel(srcPath_) - 1

	if slevel > 0 {
		paths := util.Times("..", slevel)
		paths = append(paths, destPath_)
		return fpath.Join(paths...)
	}
	if destPath_ == "/" {
		return "."
	}
	return strings.TrimPrefix(destPath_, "/")
}

func createFuncMap(curPath string) template.FuncMap {
	return template.FuncMap{
		"url": func(path string) string {
			return relativizePath(curPath, path)
		},
		"urlfor": func(id string) string {
			if env, ok := index[id]; ok {
				return relativizePath(curPath, env.Get("path"))
			}
			return "#nope"
		},
		"with_env": func(key string, value interface{}) []interface{} {
			var envs []interface{}
			for _, env := range index {
				v := env.Get(key)
				if value == v {
					envs = append(envs, env)
				}
			}
			return envs
		},
	}
}
