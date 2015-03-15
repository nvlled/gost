package main

import (
	"github.com/nvlled/gost/util"
	"os"
	fpath "path/filepath"
	"strings"
	"text/template"
)

func isValidBuildDir(dir string) bool {
	if _, err := os.Lstat(dir); err != nil {
		if os.IsNotExist(err) {
			return true
		}
	}
	_, err := os.Open(fpath.Join(dir, MARKER_NAME))
	return err == nil
}

func subDirList(baseDir string, path string) []string {
	sep := string(fpath.Separator)
	dirs := strings.Split(path, sep)

	result := []string{baseDir}
	for _, dir := range dirs {
		result = append(result, fpath.Join(baseDir, dir))
	}
	return result
}

func globTemplates(t *template.Template, key, dir string) {
	if !util.DirExists(dir) {
		println("**", key, dir, "not found")
	} else if !util.IsDirEmpty(dir) {
		_, err := t.ParseGlob(fpath.Join(dir, "*.html"))
		if err != nil {
			panic(err)
		}
	}
}

func fail(err error) {
	if err != nil {
		panic(err)
	}
}
