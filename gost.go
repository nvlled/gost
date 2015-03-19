package main

import (
	"fmt"
	//"github.com/nvlled/gost/defaults"
	"errors"
	"github.com/nvlled/gost/genv"
	"github.com/nvlled/gost/util"
	"gopkg.in/fsnotify.v1"
	"io/ioutil"
	"log"
	"os"
	fpath "path/filepath"
	"strings"
	"text/template"
)

const (
	// distdel: directory is safe to delete
	MARKER_NAME = ".gost-distdel"
)

type Index map[string]genv.T

// note: index is not actually used
var index Index
var pathIndex Index

var srcDirSet = newValidator(isSet("srcDir"), "source directory required")
var destDirSet = newValidator(isSet("destDir"), "destination directory required")
var srcDirExists = newValidator(dirExistsVar("srcDir"), "source directory does not exists")
var srcDestDiff = newValidator(notEqual("srcDir", "destDir"), "source and destination must be different")

var fullCheck = []validator{srcDirSet, srcDirExists, destDirSet, srcDestDiff}

var actions = map[string]func(*gostOpts, []string){
	"build": func(opts *gostOpts, _ []string) {
		validateOpts(opts, fullCheck...)
		state := optsToState(opts)
		runBuild(state)
	},
	"watch": func(opts *gostOpts, _ []string) {
		validateOpts(opts, fullCheck...)
		state := optsToState(opts)
		runBuild(state)
		srcDir := state.srcDir

		printLog("watching", srcDir)
		watcher, err := fsnotify.NewWatcher()
		util.RecursiveWatch(watcher, srcDir)
		fail(err)
		rebuild := util.Throttle(func() { runBuild(state) }, 900)
		for {
			select {
			case e := <-watcher.Events:
				printLog(">", e.String())
				rebuild()
			}
		}
	},
	"clean": func(opts *gostOpts, _ []string) {
		validateOpts(opts, fullCheck...)
		state := optsToState(opts)
		cleanBuildDir(state)
	},
	"newfile": func(opts *gostOpts, args []string) {
		validateOpts(opts, srcDirSet, srcDirExists)
		state := optsToState(opts)
		defer errHandler()
		if len(args) < 2 {
			println("missings args: " + args[0] + " <path> [title]")
			println("Note: path must be relative to source directory:", state.srcDir)
			return
		}
		path := args[1]
		var title string
		if len(args) > 2 {
			title = args[2]
		}
		makeFile(state, path, title)
	},
}

func runBuild(state *gostState) {
	defer errHandler()

	index = make(Index)
	pathIndex = make(Index)

	printLog("building index...")
	buildIndex(state, state.srcDir, genv.New())

	t := createTemplate()
	printLog("loading includes", state.includesDir)
	globTemplates(t, "includes-dir", state.includesDir)

	printLog("loading layouts", state.layoutsDir)
	globTemplates(t, "layouts-dir", state.layoutsDir)

	printLog("building output...", state.layoutsDir)
	buildOutput(state, t)
	println("** done.")
}

func makeFile(state *gostState, path, title string) {
	srcDir := state.srcDir
	fullpath := fpath.Join(srcDir, path)
	fulldir := fpath.Dir(fullpath)

	if info, err := os.Lstat(fullpath); err == nil {
		if info.IsDir() {
			println("file is a directory:", fullpath)
		} else {
			println("file already exists:", fullpath)
		}
		return
	}

	if _, err := os.Lstat(fulldir); os.IsNotExist(err) {
		println("directory does not exist:", fulldir)
		return
	}

	env := genv.New()
	for _, dir := range subDirList(srcDir, path) {
		parentEnv := genv.ReadDir(dir)
		env = genv.Merge(env, parentEnv)
	}
	if title != "" {
		env["title"] = title
	}

	templName := env.Get("template")
	templDir := state.templatesDir

	if templName == "" {
		println("no template for file", fullpath)
		println("add `template: the-template-name` in env")
		return
	}

	t := createTemplate()
	t.Delims("[[", "]]")
	globTemplates(t, "templates-dir", templDir)

	t = t.Lookup(templName)
	if t == nil {
		println("template not found:", templName)
		return
	}
	file, err := os.Create(fullpath)
	fail(err)
	printLog("using", "`"+templName+"`", "template from", templDir)
	err = t.ExecuteTemplate(file, templName, env)
	printLog("file created ->", fullpath)
	fail(err)
}

func cleanBuildDir(state *gostState) {
	srcDir := state.srcDir
	destDir := state.destDir

	if isValidBuildDir(destDir) {
		printLog("cleaning", destDir)
		os.RemoveAll(destDir)
		return
	}

	dirs, err := util.ReadDir(destDir, func(path string) bool {
		path = fpath.Join(srcDir, strings.TrimPrefix(path, destDir))
		if _, err := os.Lstat(path); err == nil {
			return false
		}
		return true
	})
	if err != nil {
		panic(err)
	}
	for _, dir := range dirs {
		dir = fpath.Join(destDir, dir)
		if fpath.Clean(dir) == fpath.Clean(srcDir) {
			println("** error: cannot clean source directory", srcDir, "...skipping")
			continue
		}
		printLog("removing", dir)
		os.RemoveAll(dir)
	}
}

func buildIndex(state *gostState, path string, parentEnv genv.T) {
	srcDir := state.srcDir

	info, err := os.Lstat(path)
	if err != nil {
		log.Println(err)
	} else if info.IsDir() {
		env := genv.ReadDir(path)
		env = genv.Merge(env, parentEnv)

		dirs, err := util.ReadDir(path, func(f string) bool {
			return state.isFileExcluded(f)
		})
		if err != nil {
			log.Println(err)
		} else {
			for _, name := range dirs {
				subpath := fpath.Join(path, name)
				buildIndex(state, subpath, env)
			}
		}
	} else if isItemplate(path) {
		env := genv.ReadEnv(path)
		env = genv.Merge(env, parentEnv)
		env["path"] = fpath.Join("/", strings.TrimPrefix(path, srcDir))

		pathIndex[path] = env
		if id, ok := env.GetOk("id"); ok {
			if otherEnv, dokie := index[id]; dokie {
				otherPath := otherEnv["path"]
				log.Println("Duplicate id for paths", path, otherPath)
			}
			printLog("adding", path, "to index, id =", id)
			index[id] = env
		} else {
			printLog("omitting", path, "from index (no id)")
		}
	}
}

func buildOutput(state *gostState, t *template.Template) {
	srcDir := state.srcDir
	destDir := state.destDir
	if isValidBuildDir(destDir) {
		printLog("cleaning", destDir)
		os.RemoveAll(destDir)
		util.Mkdir(destDir)

		_, err := os.Create(fpath.Join(destDir, MARKER_NAME))
		fail(err)
	}

	fn := func(srcPath string, info os.FileInfo, _ error) (err error) {
		if state.isFileExcluded(srcPath) || info.IsDir() {
			return
		}

		s := strings.TrimPrefix(srcPath, srcDir)
		destPath := fpath.Join(destDir, s)
		util.Mkdir(fpath.Dir(destPath))

		if strings.HasPrefix(destPath, srcDir) {
			println("** warning, writing to source directory")
			println("** skipping file:", destPath)
			return
		}

		env := pathIndex[srcPath]
		if isItemplate(srcPath) && !state.isFileVerbatim(s) {
			s := genv.ReadContents(srcPath)
			s = applyTemplate(t, s, env)

			if fpath.Ext(srcPath) == ".html" {
				s = applyLayout(t, s, env)
			}

			printLog("rendering", srcPath, "->", destPath)
			err = ioutil.WriteFile(destPath, []byte(s), 0644)
		} else {
			printLog("copying", srcPath, "->", destPath)
			err = util.CopyFile(destPath, srcPath)
		}
		return
	}
	fpath.Walk(srcDir, fn)
}

func errHandler() {
	err := recover()
	if err != nil {
		fmt.Printf("*** error %v\n", err)
	}
}

func isItemplate(path string) bool {
	ext := fpath.Ext(path)
	for _, ext_ := range itemplates {
		if ext == ext_ {
			return true
		}
	}
	return false
}
