package main

import (
	"github.com/nvlled/gost/util"
	fpath "path/filepath"
	"strings"
)

// TODO: move to a separate sub-package

type Vars func(string) string
type predicate func(Vars, string) bool

func (p predicate) apply(vars Vars) bool {
	return p(vars, "")
}

var NoVars = func(_ string) string { return "" }

func isSet(name string) predicate {
	return func(vars Vars, _ string) bool {
		return vars(name) != ""
	}
}

func notEqual(s1, s2 string) predicate {
	return func(vars Vars, _ string) bool {
		return vars(s1) != vars(s2)
	}
}

func isDotFile(_ Vars, path string) bool {
	return strings.HasPrefix(fpath.Base(path), ".")
}

func pathIs(path string) predicate {
	return func(_ Vars, path_ string) bool {
		return path == path_
	}
}

// potentially has a bug when given a path without a trailing slash
// since "something" is a prefix of "something-insidious"
// TODO: fix when I feel like it
func subPathIs(subpath string) predicate {
	return func(_ Vars, path string) bool {
		return strings.HasPrefix(path, subpath)
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

func dirExistsVar(name string) predicate {
	return func(vars Vars, _ string) bool {
		return util.DirExists(vars(name))
	}
}

func predicateList(newPred func(string) predicate, paths []string) []predicate {
	var preds []predicate
	for _, path := range paths {
		preds = append(preds, newPred(path))
	}
	return preds
}
