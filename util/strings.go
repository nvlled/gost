package util

import (
	"math/rand"
	fpath "path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func CommonSubPath(s1, s2 string) string {
	if s1 == "" && s2 == "" {
		return ""
	}
	sep := string(fpath.Separator)
	sub1 := strings.Split(fpath.Dir(s1), sep)
	sub2 := strings.Split(fpath.Dir(s2), sep)

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

func AddTrailingSlash(path string) string {
	if path == "/" {
		return path
	}
	sep := fpath.Separator
	return fpath.Clean(path) + string(sep)
}

func RandomString() string {
	return strconv.FormatInt(rand.Int63(), 36)
}

func RelativizePath(srcPath, destPath string) string {
	re := regexp.MustCompile(`^/`)
	if srcPath == "/" {
		if destPath == "/" {
			return "."
		}
		return strings.TrimPrefix(destPath, "/")
	}
	if !re.MatchString(destPath) {
		return destPath
	}
	if !re.MatchString(srcPath) {
		srcPath = fpath.Join("/", srcPath)
	}

	sep := string(fpath.Separator)
	prefix := CommonSubPath(destPath, srcPath) + sep

	srcPath_ := strings.TrimPrefix(srcPath, prefix)
	destPath_ := strings.TrimPrefix(destPath, prefix)

	slevel := DirLevel(srcPath_) - 1

	if slevel > 0 {
		paths := Times("..", slevel)
		paths = append(paths, destPath_)
		return fpath.Join(paths...)
	}
	if destPath_ == "/" {
		return "."
	}
	return strings.TrimPrefix(destPath_, "/")
}
