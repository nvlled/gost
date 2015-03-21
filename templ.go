package main

import (
	"bytes"
	"github.com/nvlled/gost/genv"
	"github.com/nvlled/gost/util"
	"text/template"
)

var globalFuncMap = template.FuncMap{
	"genid": util.GenerateId,
	"shell": util.Exec,

	// These stub functions are included
	// in the funcMap so that includes and layouts
	// will know these functions exists.
	// They will subsequently be overriden
	// by actual implementations created by createFuncMap()
	// which are used by applyTemplate and applyLayout.
	"url":      func(_ ...interface{}) interface{} { return "" },
	"urlfor":   func(_ ...interface{}) interface{} { return "" },
	"with_env": func(_ ...interface{}) interface{} { return "" },
}

func createFuncMap(curPath string, relativeUrl bool) template.FuncMap {
	return template.FuncMap{
		"url": func(path string) string {
			if relativeUrl {
				return util.RelativizePath(curPath, path)
			}
			return path
		},
		"urlfor": func(id string) string {
			if env, ok := index[id]; ok {
				path := env.Get("path")
				if relativeUrl {
					return util.RelativizePath(curPath, path)
				}
				return path
			}
			return "#nope"
		},
		"with_env": func(key string, value interface{}) []interface{} {
			var envs []interface{}
			for _, env := range index {
				v := env.Get(key)
				if value == v {
					envs = append(envs, env.Entries())
				}
			}
			return envs
		},
	}
}

func createTemplate() *template.Template {
	return template.New("default").Funcs(globalFuncMap)
}

func applyTemplate(t *template.Template, s string, env genv.T) string {
	curPath := env.Get("path")
	buf := new(bytes.Buffer)
	funcs := createFuncMap(curPath, isUrlRelative(env))
	entries := env.Entries()
	err := template.Must(t.New(curPath).Funcs(funcs).Parse(s)).Execute(buf, entries)
	fail(err)
	return buf.String()
}

func applyLayout(t *template.Template, s string, env genv.T) string {
	layout := env.Get(layoutKey)
	if layout == "" {
		return s
	}

	env.Set("Contents", s)
	env.Set("contents", s)
	env.Set("Body", s)
	env.Set("body", s)

	curPath := env.Get("path")
	buf := new(bytes.Buffer)
	funcs := createFuncMap(curPath, isUrlRelative(env))
	entries := env.Entries()
	err := t.New(curPath).Funcs(funcs).ExecuteTemplate(buf, layout, entries)
	fail(err)
	return buf.String()
}

func isUrlRelative(env genv.T) bool {
	if v, ok := env.GetOk(relativeKey); ok {
		return !(v == "false" || v == "0")
	}
	return true
}
