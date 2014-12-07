
package genv

import (
    "os"
    "path/filepath"
    "io/ioutil"
    "strings"
    "log"
    "fmt"
)

type T map[string]interface{}

func (env T) GetOk(k string) (string, bool) {
    v, ok := env[k]
    if !ok {
        return "", false
    }
    switch t := v.(type) {
    case string: return t, true
    default: return fmt.Sprintf("%v", t), true
    }
}

func (env T) Get(k string) string {
    v, _ := env.GetOk(k)
    return v
}

const (
    FILENAME = "env"
    SEP = ":"
    LINE_SEP = "-----"
)

func Merge(dest T, src T) T {
    env := make(T)
    for k, v := range src {
        env[k] = v
    }
    for k, v := range dest {
        if v != "" {
            env[k] = v
        }
    }
    return env
}

func Parse(s string) T {
    env := make(T)
    for _, line := range breakLines(s) {
        sub := strings.SplitN(line, SEP, 2)
        if len(sub) == 2 {
            k := strings.TrimSpace(sub[0])
            v := strings.TrimSpace(sub[1])
            env[k] = v
        }
    }
    return env
}

func ReadDir(path string) T {
    filename := filepath.Join(path, FILENAME)

    file, err := os.Open(filename)
    if err != nil { return make(T) }
    bytes, err := ioutil.ReadAll(file)
    if err != nil { return make(T) }

    return Parse(string(bytes))
}

func Read(path string) T {
    // TODO: reduce boilerplate
    file, err := os.Open(path)
    if err != nil { return make(T) }
    bytes, err := ioutil.ReadAll(file)
    if err != nil { return make(T) }

    lines := strings.Split(string(bytes), "\n")
    start := -1
    end   := -1
    for i, line := range lines {
        if line == LINE_SEP {
            if start < 0 {
                start = i+1
            } else {
                end = i
                break
            }
        }
    }
    if start < 0 || end < 0 {
        return make(T)
    }
    lines = lines[start:end]
    return Parse(joinLines(lines))
}

func ReadFile(path string) (s string) {
    file, err := os.Open(path)
    if err != nil {
        log.Println(err)
        return
    }
    bytes, err := ioutil.ReadAll(file)
    if err != nil {
        log.Println(err)
        return
    }

    lines := breakLines(string(bytes))
    end := false
    i := -1
    for j, line := range lines {
        if line == LINE_SEP {
            if end {
                i = j
                break
            } else {
                end = true
            }
        }
    }

    if i > 0 {
        s = joinLines(lines[i+1:])
    }
    return
}

func breakLines(s string) []string {
    return strings.Split(s, "\n")
}

func joinLines(lines []string) string {
    return strings.Join(lines, "\n")
}

func join(path ...string) string {
    return filepath.Join(path...)
}
