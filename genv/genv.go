package genv

import (
	"fmt"
	"github.com/nvlled/gost/util"
	"io/ioutil"
	"log"
	"os"
	fpath "path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	FILENAME = "env"
	SEP      = ":"
	LINE_SEP = "---"
)

// Since I want to be able to do something
// like {{.someValue}} using an env as a context,
// I need to use maps[string]interface{}.
// Unfortunately, this means I can't use more
// efficient datastructures for representing envs.

type T interface {
	Set(k string, v interface{})
	SetParent(T)
	GetOk(k string) (string, bool)
	Get(k string) string
	GetOr(k, defValue string) string
	Entries() map[string]interface{}
	String() string
}

type genv struct {
	entries  map[string]interface{}
	parent   T
	buffered bool
}

func newGenv() *genv {
	return &genv{
		entries:  make(map[string]interface{}),
		buffered: false,
		parent:   nil,
	}
}

func New() T {
	env := newGenv()
	return env
}

func (env *genv) New() T {
	subEnv := newGenv()
	subEnv.parent = env
	return subEnv
}

func (env *genv) getOk(k string) (string, bool) {
	v, ok := env.entries[k]
	if !ok {
		return "", false
	}
	switch t := v.(type) {
	case string:
		return t, true
	default:
		return fmt.Sprintf("%v", t), true
	}
}

func (env *genv) GetOk(k string) (string, bool) {
	if v, ok := env.getOk(k); ok {
		return v, ok
	}
	if env.parent != nil {
		return env.parent.GetOk(k)
	}
	return "", false
}

func (env *genv) Get(k string) string {
	v, _ := env.GetOk(k)
	return v
}

func (env *genv) GetOr(k string, defValue string) string {
	if val := env.Get(k); val != "" {
		return val
	}
	return defValue
}

func (env *genv) Set(k string, v interface{}) {
	env.entries[k] = v
}

func (env *genv) SetParent(parent T) {
	env.parent = parent
	env.buffered = false
}

func (env *genv) Entries() map[string]interface{} {
	if env.buffered {
		return env.entries
	}
	entries := make(map[string]interface{})
	if env.parent != nil {
		for k, v := range env.parent.Entries() {
			entries[k] = v
		}
	}
	for k, v := range env.entries {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		entries[k] = v
	}
	env.entries = entries
	env.buffered = true
	return entries
}

func (env *genv) String() string {
	entries := env.Entries()
	var keys []string
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	output := ""
	for _, k := range keys {
		output += k + SEP + " " + fmt.Sprintf("%v", entries[k]) + "\n"
	}
	return output
}

func Parse(s string) T {
	env := newGenv()
	for _, line := range strings.Split(s, "\n") {
		sub := strings.SplitN(line, SEP, 2)
		if len(sub) == 2 {
			k := strings.TrimSpace(sub[0])
			v := strings.TrimSpace(sub[1])
			env.entries[k] = v
		}
	}
	return env
}

func ReadAll(baseDir, path string) T {
	env := New()
	for _, dir := range util.SubDirList(baseDir, path) {
		subEnv := ReadDir(dir)
		subEnv.SetParent(env)
		env = subEnv
	}
	if subEnv, err := ReadFile(fpath.Join(baseDir, path)); err == nil {
		subEnv.SetParent(env)
		env = subEnv
	}
	return env
}

func ReadFile(filename string) (T, error) {
	file, err := os.Open(filename)
	if err != nil {
		return newGenv(), err
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return newGenv(), err
	}

	return Parse(string(bytes)), nil
}

func ReadDir(dir string) T {
	filename := fpath.Join(dir, FILENAME)
	env, _ := ReadFile(filename)
	return env
}

// TODO: Rename to ReadEmbedded
// expects two LINE_SEPs from a file
func ReadEnv(path string) T {
	// TODO: reduce boilerplate
	file, err := os.Open(path)
	if err != nil {
		return newGenv()
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return newGenv()
	}

	lines := strings.Split(string(bytes), "\n")
	start, end := findEnvRange(lines)
	if start < 0 || end < 0 {
		return newGenv()
	}
	lines = lines[start:end]
	return Parse(strings.Join(lines, "\n"))
}

func ReadContents(path string) string {
	file, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return ""
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		return ""
	}

	lines := strings.Split(string(bytes), "\n")
	_, end := findEnvRange(lines)
	if end > 0 {
		lines = lines[end+1:]
	}
	return strings.Join(lines, "\n")
}

// includes indices of LINE_SEP
func findEnvRange(lines []string) (int, int) {
	i := 0
	for j, line := range lines {
		if strings.TrimSpace(line) != "" {
			i = j
			break
		}
	}
	var c string
	if len(LINE_SEP) > 0 {
		c = string(LINE_SEP[0])
	}
	re := regexp.MustCompile("^" + LINE_SEP + c + "*$")
	if !re.MatchString(lines[i]) {
		return -1, -1
	}
	lineSep := lines[i]
	for j, line := range lines[i+1:] {
		if line == lineSep {
			return i, i + j + 1
		}
	}
	return -1, -1
}
