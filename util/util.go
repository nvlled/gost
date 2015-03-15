package util

import (
	"bytes"
	"fmt"
	"os/exec"
	fpath "path/filepath"
	"strings"
	"time"
)

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func DirLevel(path string) int {
	path = fpath.Join("/", path)
	return strings.Count(path, "/")
}

func Times(s string, n int) (out []string) {
	for i := 0; i < n; i++ {
		out = append(out, s)
	}
	return
}

func Throttle(action func(), millis int) func() {
	var update bool
	go func() {
		c := time.Tick(time.Duration(millis) * time.Millisecond)
		for _ = range c {
			if update {
				action()
				update = false
			}
		}
	}()

	return func() {
		update = true
	}
}

func Exec(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		return fmt.Sprintf("`%v`", err)
	}
	return strings.TrimSpace(buf.String())
}

func GenerateId() string {
	return RandomString()[:5]
}
