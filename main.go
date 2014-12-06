
package main

import (
    "os"
    "path/filepath"
    "path"
    "log"
    "strings"
    "html/template"
    "io/ioutil"
    "bytes"
    "./defaults"
    "./util"
    "./genv"
    "fmt"
    "flag"
)

const (
    MARKER_NAME = ".gost-build"
)

type Index map[string]genv.T

var index = make(Index)
var pathIndex = make(Index)

var includesDir = path.Clean(defaults.INCLUDES_DIR)
var layoutsDir = path.Clean(defaults.LAYOUTS_DIR)

var baseEnv = genv.T{
    "includes-dir" : defaults.INCLUDES_DIR,
    "layouts-dir" : defaults.LAYOUTS_DIR,
}

var funcMap = template.FuncMap{
    "absurl" : func(id string) string {
        if env, ok := index[id]; ok {
            return env.Get("path")
        }
        return "#"
    },
    "relurl" : func(curPath, id string) string {
        if env, ok := index[id]; ok {
            path := env.Get("path")
            sep := string(filepath.Separator)
            prefix := util.CommonSubPath(curPath, path)+sep

            path = strings.TrimPrefix(path, prefix)
            curPath = strings.TrimPrefix(curPath, prefix)

            n := util.DirLevel(path)
            m := util.DirLevel(curPath)

            if m > n {
                paths := util.Times("..", m - n)
                paths = append(paths, path)
                return join(paths...)
            } else {
                return path
            }
        }
        return "#"
    },
    "with_env" : func(key, val string) []string {
        return nil
    },
}

var srcDir string
var destDir string
var showHelp bool
var verbose bool

func main() {
    defer errHandler()

    parseArgs()
    prog := os.Args[0]
    if len(os.Args) < 2 || showHelp {
        println(prog, "usage:")
        flag.PrintDefaults()
        return
    }
    if !validateArgs() {
        return
    }

    env := genv.ReadDir(srcDir)
    baseEnv = genv.Merge(env, baseEnv)
    includesDir = path.Clean(join(srcDir, baseEnv.Get("includes-dir")))
    layoutsDir = path.Clean(join(srcDir, baseEnv.Get("layouts-dir")))

    printLog("building index...")
    buildIndex(srcDir, baseEnv)

    t := template.New("default").Funcs(funcMap)
    printLog("loading includes", includesDir)
    loadIncludes(t, includesDir)
    printLog("loading layouts", layoutsDir)
    loadLayouts(t, layoutsDir)
    printLog("building output...", layoutsDir)
    buildOutput(t, srcDir, destDir)
    println("** done.")
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
    if !isValidBuildDir(destDir) {
        fmt.Printf("* %s is not a valid build directory. \n", destDir)
        fmt.Printf("* Must be a non-existent directory or a previous build directory\n")
        return false
    }
    return true
}

func parseArgs() {
    flag.StringVar(&srcDir, "src", "", "source files")
    flag.StringVar(&destDir, "dest", "", "destination files")
    flag.BoolVar(&showHelp, "help", false, "show help")
    flag.BoolVar(&verbose, "verbose", true, "show verbose output")
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
        dir == layoutsDir
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
        env := genv.Read(path)
        env = genv.Merge(env, parentEnv)
        env["path"] = path

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
    path := env.Get("path")
    buf := new(bytes.Buffer)
    template.Must(t.New(path).Parse(s)).Execute(buf, env)
    return buf.String()
}

func applyLayout(t *template.Template, s string, env genv.T) string {
    layout := env.Get("layout")
    if layout == "" {
        return s
    }
    env["Contents"] = template.HTML(s)
    buf := new(bytes.Buffer)
    err := t.ExecuteTemplate(buf, layout, env); fail(err)
    return buf.String()
}

func loadIncludes(t *template.Template, dir string) {
    t.ParseGlob(join(dir, "*.html"))
}

func loadLayouts(t *template.Template, dir string) {
    _, err:= t.ParseGlob(join(dir, "*.html"))
    fail(err)
}

func buildOutput(t *template.Template, srcDir, destDir string) {
    os.RemoveAll(destDir)
    util.Mkdir(destDir)
    writeMarker(destDir)

    fn := func(srcPath string, info os.FileInfo, _ error) (err error) {
        if skipFile(srcPath) || info.IsDir() {
            return
        }

        destPath := join(destDir, srcPath)
        util.Mkdir(filepath.Dir(destPath))

        if isItemplate(srcPath) {
            env := pathIndex[srcPath]
            s := genv.ReadFile(srcPath)
            s = applyTemplate(t, s, env)
            s = applyLayout(t, s, env)
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
