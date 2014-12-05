
package util

import (
    "os"
    "io"
    "path/filepath"
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
