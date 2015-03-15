package main

import (
	"bytes"
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
	"make": func(args []string) {
		defer errHandler()
		if len(args) == 0 {
			println("missings args: make <path> [title]")
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
	loadIncludes(t, includesDir)
	printLog("loading layouts", layoutsDir)
	loadLayouts(t, layoutsDir)
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

	env := make(genv.T)
	for _, dir := range subDirList(srcDir, fpath.Dir(path)) {
		parentEnv := genv.ReadDir(dir)
		env = genv.Merge(env, parentEnv)
	}
	if title != "" {
		env["title"] = title
	}

	templName := env.Get("template")
	templDir := fpath.Join(srcDir, env.Get("templates-dir"))

	if templName == "" {
		println("no template for file", fullpath)
		println("add `template: the-template-name` in env")
		return
	}

	t := createTemplate()
	t.Delims("[[", "]]")
	loadMakeTemplates(t, templDir)

	file, err := os.Create(fullpath)
	fail(err)
	t = t.Lookup(templName)
	if t == nil {
		println("template not found:", templName)
		return
	}
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
		writeMarker(destDir)
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

func applyTemplate(t *template.Template, s string, env genv.T) string {
	curPath := env.Get("path")
	buf := new(bytes.Buffer)
	funcs := createFuncMap(curPath)
	err := template.Must(t.New(curPath).Funcs(funcs).Parse(s)).Execute(buf, env)
	fail(err)
	return buf.String()
}

func applyLayout(t *template.Template, s string, env genv.T) string {
	layout := env.Get("layout")
	if layout == "" {
		return s
	}

	env["Contents"] = s
	env["contents"] = s
	env["Body"] = s
	env["body"] = s

	curPath := env.Get("path")
	buf := new(bytes.Buffer)
	funcs := createFuncMap(curPath)
	err := t.New(curPath).Funcs(funcs).ExecuteTemplate(buf, layout, env)
	fail(err)
	return buf.String()
}

func loadIncludes(t *template.Template, dir string) {
	globTemplates(t, "includes-dir", dir)
}

func loadLayouts(t *template.Template, dir string) {
	globTemplates(t, "layouts-dir", dir)
}

func loadMakeTemplates(t *template.Template, dir string) {
	globTemplates(t, "templates-dir", dir)
}
