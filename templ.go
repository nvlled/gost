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
}

func createFuncMap(curPath string) template.FuncMap {
	return template.FuncMap{
		"url": func(path string) string {
			return util.RelativizePath(curPath, path)
		},
		"urlfor": func(id string) string {
			if env, ok := index[id]; ok {
				return util.RelativizePath(curPath, env.Get("path"))
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

func applyTemplate(t *template.Template, s string, env genv.T) string {
	curPath := env.Get("path")
	buf := new(bytes.Buffer)
	funcs := createFuncMap(curPath)
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
	funcs := createFuncMap(curPath)
	entries := env.Entries()
	err := t.New(curPath).Funcs(funcs).ExecuteTemplate(buf, layout, entries)
	fail(err)
	return buf.String()
}

func createTemplate() *template.Template {
	funcMap := createFuncMap(".")
	for k, v := range globalFuncMap {
		funcMap[k] = v
	}

	return template.New("default").Funcs(funcMap)
}
