
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
    "fmt"
)

type Index map[string]Env

var index = make(Index)
var pathIndex = make(Index)

var includesDir = path.Clean(defaults.INCLUDES_DIR)
var layoutsDir = path.Clean(defaults.LAYOUTS_DIR)

var baseEnv = Env{
    "includes-dir" : defaults.INCLUDES_DIR,
    "layouts-dir" : defaults.LAYOUTS_DIR,
}

var funcMap = template.FuncMap{
    "urlfor" : func(name string) string {
        return "--------"
    },
    "with_env" : func(key, val string) []string {
        return nil
    },
}

func main() {
    defer errHandler()
    srcDir  := "sample-site/"
    destDir := "build/"

    env := readDirEnv(srcDir)
    baseEnv = merge(env, baseEnv)
    includesDir = path.Clean(join(srcDir, baseEnv.get("includes-dir")))
    layoutsDir = path.Clean(join(srcDir, baseEnv.get("layouts-dir")))

    println("building index...")
    buildIndex(srcDir, baseEnv)

    t := template.New("default").Funcs(funcMap)
    println("loading includes", includesDir)
    loadIncludes(t, includesDir)
    println("loading layouts", layoutsDir)
    loadLayouts(t, layoutsDir)
    println("building output...", layoutsDir)
    buildOutput(t, srcDir, destDir)
    println("** done.")
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
        base == ENV_FILENAME ||
        dir == includesDir ||
        dir == layoutsDir
}

func buildIndex(path string, parentEnv Env) {
    info, err := os.Lstat(path)
    if err != nil {
        log.Println(err)
    } else if info.IsDir() {
        // FIX: baseEnv is read twice
        env := readDirEnv(path)
        env = merge(env, parentEnv)

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
        env := readEnv(path)
        env = merge(env, parentEnv)
        env["path"] = path

        pathIndex[path] = env
        if id, ok := env.getOk("id"); ok {
            if otherEnv, dokie := index[id]; dokie {
                otherPath := otherEnv["path"]
                log.Println("Duplicate id for paths", path, otherPath)
            }
            println("adding", path, "to index, id =", id)
            index[id] = env
        } else {
            println("omitting", path, "from index (no id)")
        }
    }
}

func applyTemplate(t *template.Template, s string, env Env) string {
    path := env.get("path")
    buf := new(bytes.Buffer)
    template.Must(t.New(path).Parse(s)).Execute(buf, env)
    return buf.String()
}

func applyLayout(t *template.Template, s string, env Env) string {
    layout := env.get("layout")
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

    fn := func(srcPath string, info os.FileInfo, _ error) (err error) {
        if skipFile(srcPath) || info.IsDir() {
            return
        }

        destPath := join(destDir, srcPath)
        util.Mkdir(filepath.Dir(destPath))

        if isItemplate(srcPath) {
            env := pathIndex[srcPath]
            s := readFile(srcPath)
            s = applyTemplate(t, s, env)
            s = applyLayout(t, s, env)
            println("rendering", srcPath, "->", destPath)
            err = ioutil.WriteFile(destPath, []byte(s), 0644)
        } else {
            println("copying", srcPath, "->", destPath)
            err = util.CopyFile(destPath, srcPath)
        }
        return
    }
    filepath.Walk(srcDir, fn)
}


func join(path ...string) string {
    return filepath.Join(path...)
}

func fail(err error) {
    if err != nil {
        panic(err)
    }
}
