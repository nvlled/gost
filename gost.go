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

// TODO: change all / to os.PathSeparator

const (
	// distdel: directory is safe to delete
	MARKER_NAME = ".gost-distdel"
)

type Index map[string]genv.T

var index Index
var pathIndex Index

var srcDirSet = newValidator(isSet("srcDir"), "source directory required")
var destDirSet = newValidator(isSet("destDir"), "destination directory required")
var srcDirExists = newValidator(dirExistsVar("srcDir"), "source directory does not exists")
var srcDestDiff = newValidator(notEqual("srcDir", "destDir"), "source and destination must be different")

var fullCheck = []validator{srcDirSet, srcDirExists, destDirSet, srcDestDiff}

type action struct {
	help string
	fn   func(*gostOpts, []string)
}

var actions = map[string]action{
	"new": action{
		util.Detab(`usage: %s %s <projectname>

		|Creates a new project based on a template.
		|The project is placed on a directory
		|named <projectname>.
		`),
		func(_ *gostOpts, args []string) {
			if len(args) < 2 {
				fmt.Printf("missing args: %s <name>\n", args[0])
				return
			}
			dirname := args[1]
			if util.DirExists(dirname) {
				println("directory already exists: ", dirname)
				return
			}
			err := newSampleProject(dirname)
			if err != nil {
				fmt.Printf("New project creation failed, %s", err)
			}
		},
	},
	"build": action{
		util.Detab(`usage: %s --srcDir <dir> --destDir <dir> %s

		|Builds the projects from srcDir and stores
		|them in destDir. srcDir and destDir may also be
		|specified in the opts file.

		|srcDir and destDir must not be the same.
		`),
		func(opts *gostOpts, _ []string) {
			validateOpts(opts, fullCheck...)
			state := optsToState(opts)
			runBuild(state)
		},
	},
	"watch": action{
		util.Detab(`usage: %s --srcDir <dir> --destDir <dir> %s

		|Same as build action, but watches the
		|srcDir for changes (such creation
		|of a new file or modification of an
		|existing file) and then re-builds the project
		|accordingly..
		`),
		func(opts *gostOpts, _ []string) {
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
	},
	"clean": action{
		util.Detab(`usage: %s --srcDir <dir> --destDir <dir> %s

		|Removes all the files created from a build action.
		|If dest is a directory created from build action,
		|as indicated by the presence of .gost-distdel in it,
		|then it is deleted.
		`),
		func(opts *gostOpts, _ []string) {
			validateOpts(opts, fullCheck...)
			state := optsToState(opts)
			cleanBuildDir(state)
		},
	},
	"newfile": action{
		util.Detab(`usage: %s --srcDir <dir> %s <filename>

		|Creates a file in the project.
		|Directory of <filename> must be relative to
		|the srcDir.
		|Example: newfile posts/hello.html

		|Also, env of the directory of <filename>
		|must contain a template entry.
		`),
		func(opts *gostOpts, args []string) {
			validateOpts(opts, srcDirSet, srcDirExists)
			state := optsToState(opts)
			defer catchError()
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
			newProjectFile(state, path, title)
		},
	},
}

func runBuild(state *gostState) {
	defer catchError()

	index = make(Index)
	pathIndex = make(Index)

	printLog("building index...")
	buildIndex(state, state.srcDir, genv.New())

	t := createTemplate()
	printLog("loading includes", state.includesDir)
	globTemplates(t, includesKey, state.includesDir)

	printLog("loading layouts", state.layoutsDir)
	globTemplates(t, layoutsKey, state.layoutsDir)

	printLog("building output...", state.layoutsDir)
	buildOutput(state, t)
	println("** done.")
}

func newProjectFile(state *gostState, path, title string) {
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

	templName := env.Get(templateKey)
	templDir := state.templatesDir

	if templName == "" {
		println("no template for file", fullpath)
		println("add `template: the-template-name` in env")
		return
	}

	t := createTemplate()
	t.Delims(templateOpenDelim, templateCloseDelim)
	globTemplates(t, templatesKey, templDir)

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

		if state.isFileExcluded(s) {
			printLog("*** skipping excluded file: " + s)
			return
		}
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

func newSampleProject(dirname string) error {
	join := fpath.Join
	srcDir := "src"
	destDir := "build"
	layoutFile := "default.html"
	templateFile := "article.html"

	if util.DirExists(dirname) {
		return errors.New("directory already exists: " + dirname)
	}
	printLog("*** creating project " + dirname)

	mkdir := func(path string) {
		printLog("create dir:  ", path)
		util.Mkdir(path)
	}
	createFile := func(path, contents string) {
		path = join(dirname, path)
		printLog("create file: ", path)
		util.CreateFile(path, contents)
	}
	detabf := func(s string, args ...interface{}) string {
		return util.Detab(fmt.Sprintf(s, args...))
	}

	mkdir(dirname)
	mkdir(join(dirname, "build"))
	mkdir(join(dirname, srcDir))
	mkdir(join(dirname, srcDir, defaultIncludesDir))
	mkdir(join(dirname, srcDir, defaultLayoutsDir))
	mkdir(join(dirname, srcDir, defaultTemplatesDir))
	mkdir(join(dirname, srcDir, "articles"))
	mkdir(join(dirname, srcDir, "sample-files"))
	mkdir(join(dirname, srcDir, "trash"))

	createFile(defaultOptsfile, detabf(`
	|--srcDir %s
	|--destDir %s`, srcDir, destDir))

	createFile(join(srcDir, genv.FILENAME), detabf(`
	|layout: %s
	|sitename: %s
	|verbatim: sample-files/
	|excludes: trash/
	`, layoutFile, dirname))

	createFile(join(srcDir, "articles", genv.FILENAME), detabf(`
	|template: %s
	|category: article`, templateFile))

	createFile(join(srcDir, defaultLayoutsDir, layoutFile), detabf(`
	|<html lang="en">
	|<head>
	|<meta charset="UTF-8">
	|<title>{{with .title}}{{.}} - {{end}}{{.sitename}}</title>
	|</head>
	|<body>
	|<div id="wrapper">
	|<a href='{{urlfor "home"}}'>home</a>
	|<h2>{{.title}}</h2>
	|{{.contents}}
	|</div>
	|</body>
	|</html>`))

	createFile(join(srcDir, defaultLayoutsDir, "other.html"), detabf(`
	|<html lang="en">
	|<body>
	|<div id="sidebar">
	|<a href='{{urlfor "home"}}'>home</a>
	|<a href='{{urlfor "hello"}}'>hello</a>
	|</div>
	|<div id="contents">
	|{{.contents}}
	|</div>
	|<div id="footer">fock semantic tags</div>
	|</body>
	|</html>`))

	createFile(join(srcDir, defaultIncludesDir, "includes.html"), detabf(`
	|{{define "emphasize"}}
	|<em><blink>__{{.}}__</blink><em>
	|{{end}}
	`))

	createFile(join(srcDir, "index.html"), detabf(`
	|-------------------------
	|id: home
	|title: Welcome
	|-------------------------

	|<p>This is the home page</p>

	|<h3>articles</h3>
	|<ul>
	|{{range (with_env "category" "article")}}
	|<li><a href="{{.path}}">{{.title}}</a></li>
	|{{end}}
	|</ul>`))

	createFile(join(srcDir, defaultTemplatesDir, templateFile), detabf(`
	|----------------------
	|id: [[genid]]
	|title: [[.title]]
	|date: [[shell "date"]]
	|----------------------

	|<p>pikachu elf is fake</p>`))

	createFile(join(srcDir, "articles", "hello.html"), detabf(`
	|--------
	|id: hello
	|title: Title is hello
	|--------

	|<p>Hello, this is a greeting with no intrinsic value
	|See the other equally-useless <a href='{{urlfor "sample"}}'>article</a>
	|{{template "emphasize" "u sock"}}
	|</p>`))

	createFile(join(srcDir, "articles", "sample.html"), detabf(`
	|--------
	|id: sample
	|title: A title
	|layout: other.html
	|--------

	|<p>A sample page with sample links</a>
	|<a href="sample-files/verbatim.html">verbatim file</a>`))

	createFile(join(srcDir, "sample-files", "verbatim.html"), detabf(`
	|<p>An html file with no layout</p>`))

	createFile(join(srcDir, "trash", "testfile"), detabf(`
	|a discarded file but not yet deleted for possible future reference
	|this will not be included in the build
	`))

	printLog("*** done")
	return nil
}
