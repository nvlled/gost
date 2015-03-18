package main

import (
	fpath "path/filepath"
	"strings"
)

type Vars func(string) string
type predicate func(Vars, string) bool

func isDotFile(_ Vars, path string) bool {
	return strings.HasPrefix(fpath.Base(path), ".")
}

func pathIs(path string) predicate {
	return func(_ Vars, path_ string) bool {
		return path == path_
	}
}

func baseIs(base string) predicate {
	return func(_ Vars, path string) bool {
		return fpath.Base(path) == base
	}
}

func baseIsVar(name string) predicate {
	return func(vars Vars, path string) bool {
		return fpath.Base(path) == vars(name)
	}
}

func dirIs(dir string) predicate {
	return func(_ Vars, path string) bool {
		return fpath.Dir(path) == dir
	}
}

func dirIsVar(name string) predicate {
	return func(vars Vars, path string) bool {
		return fpath.Dir(path) == vars(name)
	}
}

func predicateList(newPred func(string) predicate, paths []string) []predicate {
	var preds []predicate
	for _, path := range paths {
		preds = append(preds, newPred(path))
	}
	return preds
}
