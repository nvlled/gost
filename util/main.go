
package util

import (
    "os"
    "io"
    "path/filepath"
    "strings"
    "gopkg.in/fsnotify.v1"
    "time"
)

func ReadDir(path string, filter func(string)bool) ([]string, error) {
    file, err := os.Open(path)
    if err != nil { return nil, err }
    names, err := file.Readdirnames(-1)
    if err != nil { return nil, err }

    var names_ []string
    for _, name := range names {
        if !filter(filepath.Join(path, name)) {
            names_ = append(names_, name)
        }
    }
    return names_, nil
}

func CopyFile(destPath, srcPath string) (err error) {
    src, err := os.Open(srcPath)
    if err != nil { return }
    dest, err := os.Create(destPath)
    if err != nil { return }

    _, err = io.Copy(dest, src)
    return
}

func Mkdir(path string) {
    os.MkdirAll(path, os.ModeDir | 0775)
}


func CommonSubPath(s1, s2 string) string {
    if s1 == "" && s2 == "" {
        return ""
    }
    sep := string(filepath.Separator)
    sub1 := strings.Split(filepath.Dir(s1), sep)
    sub2 := strings.Split(filepath.Dir(s2), sep)

    var paths []string
    for i := 0; i < Min(len(sub1), len(sub2)); i++ {
        if sub1[i] == sub2[i] {
            paths = append(paths, sub1[i])
        }
    }
    return strings.Join(paths, sep)
}

func CommonPrefix(s1, s2 string) string {
    b1 := []byte(s1)
    b2 := []byte(s2)

    var prefix []byte
    for i := 0; i < Min(len(b1), len(b2)); i++ {
        if b1[i] == b2[i] {
            prefix = append(prefix, b1[i])
        }
    }
    return string(prefix)
}

func Max(x, y int) int {
    if x > y {
        return x
    }
    return y
}

func Min(x, y int) int {
    if x < y {
        return x
    }
    return y
}

func DirLevel(path string) int {
    path = filepath.Clean(path)
    paths := strings.SplitAfter(path, string(filepath.Separator))
    return len(paths)
}

func Times(s string, n int) (out []string) {
    for i := 0; i < n; i++ {
        out = append(out, s)
    }
    return
}

func Throttle(action func(), millis int) func() {
    var update bool
    go func() {
            c := time.Tick(time.Duration(millis) * time.Millisecond)
            for _ = range c {
                if update {
                    action()
                    update = false
                }
            }
    } ()

    return func() {
        update = true
    }
}

func RecursiveWatch(w *fsnotify.Watcher, dir string) {
    filepath.Walk(dir, func(path string, info os.FileInfo, _ error) error {
        if info.IsDir() {
            w.Add(path)
        }
        return nil
    })
}

func DirExists(path string) bool {
    info, err := os.Lstat(path)
    if err != nil {
        return false
    }
    return info.IsDir()
}

func AddTrailingSlash(path string) string {
    if (path == "/") {
        return path
    }
    sep := filepath.Separator
    return filepath.Clean(path)+string(sep)
}

func IsDirEmpty(dir string) bool {
    names, err := ReadDir(dir, func(_ string)bool { return false })
    if err != nil {
        return false
    }
    return len(names) == 0
}
