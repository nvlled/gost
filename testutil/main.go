
package testutil

import "testing"

func TestStringOp2(t *testing.T, data [][]string, fn func(string, string)string) {
    for _, row := range data {
        result := fn(row[0], row[1])
        if result != row[2] {
            t.Error("f("+row[0]+", "+row[1]+")", "Expected", row[2], "Got", result)
        }
    }
}
