
package main

import (
    "os"
    "path/filepath"
    "path"
    "log"
    "strings"
    "text/template"
    "io/ioutil"
    "bytes"
    "github.com/nvlled/gost/defaults"
    "github.com/nvlled/gost/util"
    "github.com/nvlled/gost/genv"
    "fmt"
    "flag"
    "gopkg.in/fsnotify.v1"
    "regexp"
)

const (
    MARKER_NAME = ".gost-build"
    VERBATIM_KEY = "verbatim"
)

type Index map[string]genv.T

var index Index
var pathIndex Index

var includesDir = path.Clean(defaults.INCLUDES_DIR)
var layoutsDir = path.Clean(defaults.LAYOUTS_DIR)

var baseEnv = genv.T{
    "includes-dir" : defaults.INCLUDES_DIR,
    "layouts-dir" : defaults.LAYOUTS_DIR,
}

var srcDir string
var destDir string
var buildFile string
var showHelp bool
var verbose bool
var watchSource bool
var verbatimList []string
var cleanBuildFiles bool

func main() {
    defer errHandler()

    parseArgs()
    if buildFile != "" {
        readBuildFile(buildFile)
    }

    prog := os.Args[0]
    if showHelp || (len(os.Args) < 2 && srcDir == "") {
        println(prog, "usage:")
        flag.PrintDefaults()
        return
    }

    if !validateArgs() {
        return
    }

    srcDir = util.AddTrailingSlash(srcDir)
    destDir = util.AddTrailingSlash(destDir)

    if cleanBuildFiles {
        cleanBuildDir(srcDir, destDir)
        return
    }

    run := func() {
        index = make(Index)
        pathIndex = make(Index)
        env := genv.ReadDir(srcDir)
        baseEnv = genv.Merge(env, baseEnv)

        includesDir = path.Clean(join(srcDir, baseEnv.Get("includes-dir")))
        layoutsDir = path.Clean(join(srcDir, baseEnv.Get("layouts-dir")))
        verbatimList = strings.Fields(baseEnv.Get(VERBATIM_KEY))

        printLog("building index...")
        buildIndex(srcDir, baseEnv)

        t := template.New("default").Funcs(createFuncMap("."))
        printLog("loading includes", includesDir)
        loadIncludes(t, includesDir)
        printLog("loading layouts", layoutsDir)
        loadLayouts(t, layoutsDir)
        printLog("building output...", layoutsDir)
        buildOutput(t, srcDir, destDir)
        println("** done.")
    }

    run()

    if watchSource {
        printLog("watching", srcDir)
        watcher, err := fsnotify.NewWatcher()
        util.RecursiveWatch(watcher, srcDir)
        fail(err)
        run = util.Throttle(run, 900)
        for {
            select {
            case e := <-watcher.Events:
                printLog(">", e.String())
                run()
            }
        }
    }
}

func validateArgs() bool {
    info, err := os.Lstat(srcDir)
    if err != nil {
        fmt.Printf("failed to open directory: %s\n", srcDir)
        return false
    }
    if !info.IsDir() {
        fmt.Printf("%s is not a directory\n", srcDir)
        return false
    }
    if destDir == "" {
        fmt.Printf("destination directory required\n")
        return false
    }
    if srcDir == destDir {
        fmt.Printf("source and destination must not be the same\n")
        return false
    }
    return true
}

func parseArgs() {
    flag.StringVar(&srcDir, "src", "", "source files")
    flag.StringVar(&destDir, "dest", "", "destination files")
    flag.StringVar(&buildFile, "build", "gost-build", "build file")
    flag.BoolVar(&showHelp, "help", false, "show help")
    flag.BoolVar(&verbose, "verbose", true, "show verbose output")
    flag.BoolVar(&watchSource, "watch", false, "watch source directory")
    flag.BoolVar(&cleanBuildFiles, "clean", false, "clean build dir")
    flag.Parse()
}

func errHandler() {
    err := recover()
    if err != nil {
        fmt.Printf("*** error %v\n", err)
    }
}

func isItemplate(path string) bool {
    ext := filepath.Ext(path)
    return ext == ".html" ||
           ext == ".js"   ||
           ext == ".css"

}

func skipFile(file string) bool {
    base := filepath.Base(file)
    dir := filepath.Dir(file)
    return strings.HasPrefix(base, ".") || //exclude dot files
        base == genv.FILENAME ||
        dir == includesDir ||
        dir == layoutsDir ||
        base == MARKER_NAME
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
                subpath := join(path, name)
                buildIndex(subpath, env)
            }
        }
    } else if isItemplate(path) {
        env := genv.ReadEnv(path)
        env = genv.Merge(env, parentEnv)
        env["path"] = strings.TrimPrefix(path, srcDir)

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
    if !util.DirExists(dir) {
        println("** includes-dir", dir, "not found")
    } else if !util.IsDirEmpty(dir) {
        _, err := t.ParseGlob(join(dir, "*.html"))
        if err != nil {
            panic(err)
        }
    }
}

func loadLayouts(t *template.Template, dir string) {
    if !util.DirExists(dir) {
        println("** layouts-dir", dir, "not found")
    } else if !util.IsDirEmpty(dir) {
        _, err := t.ParseGlob(join(dir, "*.html"))
        if err != nil {
            panic(err)
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
        destPath := join(destDir, s)
        util.Mkdir(filepath.Dir(destPath))

        env := pathIndex[srcPath]
        if isItemplate(srcPath) && !isVerbatim(env, s) {
            s := genv.ReadContents(srcPath)
            s = applyTemplate(t, s, env)

            if filepath.Ext(srcPath) == ".html" {
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
    filepath.Walk(srcDir, fn)
}

func genUrl(srcPath, destPath string) string {
    re := regexp.MustCompile(`^/`)
    if re.MatchString(destPath) {
        return destPath
    }
    if re.MatchString(srcPath) {
        return join(srcPath, destPath)
    }

    sep := string(filepath.Separator)
    prefix := util.CommonSubPath(destPath, srcPath)+sep

    srcPath_ := strings.TrimPrefix(srcPath, prefix)
    destPath_ := strings.TrimPrefix(destPath, prefix)

    dlevel := util.DirLevel(destPath_)
    slevel := util.DirLevel(srcPath_)

    if slevel >= dlevel {
        paths := util.Times("..", slevel-1)
        paths = append(paths, destPath_)
        return join(paths...)
    } else {
        return destPath
    }
}

func urlFor(curPath, id string)string {
    if env, ok := index[id]; ok {
        return genUrl(curPath, env.Get("path"))
    }
    return "#nope"
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

func readBuildFile(path string) {
    env := genv.ReadFile(path)
    fmt.Printf("%v\n", env)
    if srcDir == "" {
        srcDir = env.Get("src")
    }
    if destDir == "" {
        destDir = env.Get("dest")
    }
}

func createFuncMap(curPath string) template.FuncMap {
    return template.FuncMap{
        "url" : func(path string) string {
            return genUrl(curPath, path)
        },
        "urlfor" : func(id string) string {
            return urlFor(curPath, id)
        },
    }
}

func cleanBuildDir(srcDir, destDir string) {
    if isValidBuildDir(destDir) {
        printLog("cleaning", destDir)
        os.RemoveAll(destDir)
        return
    }

    dirs, err := util.ReadDir(destDir, func(path string) bool {
        path = join(srcDir, strings.TrimPrefix(path, destDir))
        if _, err := os.Lstat(path); err == nil {
            return false
        }
        return true
    })
    if err != nil {
        panic(err)
    }
    for _, dir := range dirs {
        dir = join(destDir, dir)
        printLog("removing", dir)
        os.RemoveAll(dir)
    }
}

func printLog(args ...interface{}) {
    if verbose {
        fmt.Println(args...)
    }
}

func isValidBuildDir(dir string) bool {
    if _, err := os.Lstat(dir); err != nil {
        if os.IsNotExist(err) {
            return true
        }
    }
    _, err := os.Open(join(dir, MARKER_NAME))
    return err == nil
}

func writeMarker(dir string) {
    _, err := os.Create(join(dir, MARKER_NAME))
    fail(err)
}

func join(path ...string) string {
    return filepath.Join(path...)
}

func fail(err error) {
    if err != nil {
        panic(err)
    }
}
