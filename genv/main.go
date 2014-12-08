
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
    LINE_SEP = "----"
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
    start, end := findEnvRange(lines)
    if start < 0 || end < 0 {
        return make(T)
    }
    lines = lines[start:end]
    return Parse(joinLines(lines))
}

func ReadFile(path string) string {
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

    lines := breakLines(string(bytes))
    _, end := findEnvRange(lines)
    if end > 0 {
        lines = lines[end+1:]
    }
    return joinLines(lines)
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
    if lines[i] != LINE_SEP {
        return -1, -1
    }
    for j, line := range lines[i+1:] {
        if line == LINE_SEP {
            return i, i+j+1
        }
    }
    return -1, -1
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
