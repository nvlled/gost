package main

import (
	"fmt"
	"github.com/nvlled/gost/defaults"
	"github.com/nvlled/gost/genv"
	"github.com/nvlled/gost/util"
	"gopkg.in/fsnotify.v1"
	"io/ioutil"
	"log"
	"os"
	"path"
	fpath "path/filepath"
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

var actions = map[string]func(args []string){
	"build": func(_ []string) {
		runBuild()
	},
	"watch": func(_ []string) {
		runBuild()
		printLog("watching", srcDir)
		watcher, err := fsnotify.NewWatcher()
		util.RecursiveWatch(watcher, srcDir)
		fail(err)
		rebuild := util.Throttle(runBuild, 900)
		for {
			select {
			case e := <-watcher.Events:
				printLog(">", e.String())
				rebuild()
			}
		}
	},
	"clean": func(_ []string) {
		cleanBuildDir(srcDir, destDir)
	},
	"newfile": func(args []string) {
		defer errHandler()
		if len(args) == 0 {
			println("missings args: newfile <path> [title]")
			println("Note: path must be relative to source directory:", srcDir)
			return
		}
		path := args[0]
		var title string
		if len(args) > 1 {
			title = args[1]
		}
		makeFile(path, title)
	},
}

func runBuild() {
	defer errHandler()

	index = make(Index)
	pathIndex = make(Index)
	env := genv.ReadDir(srcDir)
	baseEnv = genv.Merge(env, baseEnv)

	initializeDirs(baseEnv)
	verbatimList = strings.Fields(baseEnv.Get(VERBATIM_KEY))

	printLog("building index...")
	buildIndex(srcDir, baseEnv)

	t := createTemplate()
	printLog("loading includes", includesDir)
	globTemplates(t, "includes-dir", includesDir)

	printLog("loading layouts", layoutsDir)
	globTemplates(t, "layouts-dir", layoutsDir)

	printLog("building output...", layoutsDir)
	buildOutput(t, srcDir, destDir)
	println("** done.")
}

func makeFile(path, title string) {
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

	env := baseEnv
	for _, dir := range subDirList(srcDir, path) {
		parentEnv := genv.ReadDir(dir)
		env = genv.Merge(env, parentEnv)
	}
	if title != "" {
		env["title"] = title
	}

	templName := env.Get("template")
	templDir := fpath.Join(srcDir, env.Get(TEMPLATES_DIR_KEY))

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

func cleanBuildDir(srcDir, destDir string) {
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

func buildIndex(path string, parentEnv genv.T) {
	info, err := os.Lstat(path)
	if err != nil {
		log.Println(err)
	} else if info.IsDir() {
		// FIX: baseEnv is read twice
		env := genv.ReadDir(path)
		env = genv.Merge(env, parentEnv)

		dirs, err := util.ReadDir(path, skipFile)

		if err != nil {
			log.Println(err)
		} else {
			for _, name := range dirs {
				subpath := fpath.Join(path, name)
				buildIndex(subpath, env)
			}
		}
	} else if isItemplate(path) {
		env := genv.ReadEnv(path)
		env = genv.Merge(env, parentEnv)
		//env["path"] = strings.TrimPrefix(path, srcDir)
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

func buildOutput(t *template.Template, srcDir, destDir string) {
	if isValidBuildDir(destDir) {
		printLog("cleaning", destDir)
		os.RemoveAll(destDir)
		util.Mkdir(destDir)

		_, err := os.Create(fpath.Join(destDir, MARKER_NAME))
		fail(err)
	}

	fn := func(srcPath string, info os.FileInfo, _ error) (err error) {
		if skipFile(srcPath) || info.IsDir() {
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
		if isItemplate(srcPath) && !isVerbatim(env, s) {
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

func initializeDirs(env genv.T) {
	prependSrc := func(dir string) string {
		if dir != "" {
			dir = path.Clean(fpath.Join(srcDir, dir))
		}
		return dir
	}
	includesDir = prependSrc(env.Get(INCLUDES_DIR_KEY))
	layoutsDir = prependSrc(env.Get(LAYOUTS_DIR_KEY))
	templatesDir = prependSrc(env.Get(TEMPLATES_DIR_KEY))
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
