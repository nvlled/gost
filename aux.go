package main

import (
	"fmt"
	"github.com/nvlled/gost/util"
	"os"
	fpath "path/filepath"
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

func catchError() {
	err := recover()
	if err != nil {
		fmt.Printf("*** error %v\n", err)
	}
}

func isItemplate(path string) bool {
	ext := fpath.Ext(path)
	for _, ext_ := range itemplates {
		if ext == ext_ {
			return true
		}
	}
	return false
}
