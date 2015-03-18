package main

import (
	"github.com/nvlled/gost/util"
)

type gostState struct {
	srcDir       string
	destDir      string
	includesDir  string
	layoutsDir   string
	templatesDir string

	verbatimList []predicate
	excludeList  []predicate
}

func newState(srcDir, destDir string) *gostState {
	return &gostState{
		srcDir:  srcDir,
		destDir: destDir,
	}
}

func (state *gostState) setIncludesDir(dir string) *gostState {
	state.includesDir = util.PrependPath(dir, state.srcDir)
	return state
}

func (state *gostState) setLayoutsDir(dir string) *gostState {
	state.layoutsDir = util.PrependPath(dir, state.srcDir)
	return state
}

func (state *gostState) setTemplatesDir(dir string) *gostState {
	state.templatesDir = util.PrependPath(dir, state.srcDir)
	return state
}

func (state *gostState) setVerbatimList(preds []predicate) *gostState {
	state.verbatimList = preds
	return state
}

func (state *gostState) setExcludeList(preds []predicate) *gostState {
	state.excludeList = preds
	return state
}

func (state *gostState) makeVars() Vars {
	return func(s string) string {
		switch s {
		case "srcDir":
			return state.srcDir
		case "destDir":
			return state.destDir
		case "includesDir":
			return state.includesDir
		case "layoutsDir":
			return state.layoutsDir
		case "templatesDir":
			return state.templatesDir
		}
		return ""
	}
}

func (state *gostState) isFileExcluded(file string) bool {
	vars := state.makeVars()
	for _, pred := range state.excludeList {
		if pred(vars, file) {
			return true
		}
	}
	return false
}

func (state *gostState) isFileVerbatim(file string) bool {
	vars := state.makeVars()
	for _, pred := range state.verbatimList {
		if pred(vars, file) {
			return true
		}
	}
	return false
}
