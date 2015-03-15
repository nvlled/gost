package util

import (
	"gopkg.in/fsnotify.v1"
	"io"
	"os"
	fpath "path/filepath"
)

func ReadDir(path string, filter func(string) bool) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	names, err := file.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	var names_ []string
	for _, name := range names {
		if !filter(fpath.Join(path, name)) {
			names_ = append(names_, name)
		}
	}
	return names_, nil
}

func CopyFile(destPath, srcPath string) (err error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return
	}
	dest, err := os.Create(destPath)
	if err != nil {
		return
	}

	_, err = io.Copy(dest, src)
	return
}

func IsDirEmpty(dir string) bool {
	names, err := ReadDir(dir, func(_ string) bool { return false })
	if err != nil {
		return false
	}
	return len(names) == 0
}

func DirExists(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func Mkdir(path string) {
	os.MkdirAll(path, os.ModeDir|0775)
}

func RecursiveWatch(w *fsnotify.Watcher, dir string) {
	fpath.Walk(dir, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			w.Add(path)
		}
		return nil
	})
}
