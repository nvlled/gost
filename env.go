
package main

import (
    "os"
    "path/filepath"
    "io/ioutil"
    "strings"
    "log"
    "fmt"
)

type Env map[string]interface{}

func (env Env) getOk(k string) (string, bool) {
    v, ok := env[k]
    if !ok {
        return "", false
    }
    switch t := v.(type) {
    case string: return t, true
    default: return fmt.Sprintf("%v", t), true
    }
}

func (env Env) get(k string) string {
    v, _ := env.getOk(k)
    return v
}

const (
    ENV_FILENAME = "env"
    ENV_SEP = ":"
    ENV_LINE_SEP = "-----"
)

func merge(dest Env, src Env) Env {
    env := make(Env)
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

func parseEnv(s string) Env {
    env := make(Env)
    for _, line := range breakLines(s) {
        sub := strings.SplitN(line, ENV_SEP, 2)
        if len(sub) == 2 {
            k := strings.TrimSpace(sub[0])
            v := strings.TrimSpace(sub[1])
            env[k] = v
        }
    }
    return env
}

func readDirEnv(path string) Env {
    filename := filepath.Join(path, ENV_FILENAME)

    file, err := os.Open(filename)
    if err != nil { return make(Env) }
    bytes, err := ioutil.ReadAll(file)
    if err != nil { return make(Env) }

    return parseEnv(string(bytes))
}

func readEnv(path string) Env {
    // TODO: reduce boilerplate
    file, err := os.Open(path)
    if err != nil { return make(Env) }
    bytes, err := ioutil.ReadAll(file)
    if err != nil { return make(Env) }

    lines := strings.Split(string(bytes), "\n")
    start := -1
    end   := -1
    for i, line := range lines {
        if line == ENV_LINE_SEP {
            if start < 0 {
                start = i+1
            } else {
                end = i
                break
            }
        }
    }
    if start < 0 || end < 0 {
        return make(Env)
    }
    lines = lines[start:end]
    return parseEnv(joinLines(lines))
}

func readFile(path string) (s string) {
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

    s = string(bytes)
    i := strings.LastIndex(s, ENV_LINE_SEP)
    if i > 0 {
        // Skip env block
        i += len(ENV_LINE_SEP)+1
        s = string(bytes[i:])
    }
    return
}

func breakLines(s string) []string {
    return strings.Split(s, "\n")
}

func joinLines(lines []string) string {
    return strings.Join(lines, "\n")
}
