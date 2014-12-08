
package util

import (
    "testing"
    "github.com/nvlled/gost/testutil"
)

func TestCommonSubPath(t *testing.T) {
    testData := [][]string{
        //path1             path2           expected
        {"abc/efg",         "abc/xyz",      "abc"},
        {"xyz",             "abc",          "."},
        {"xyz",             "xyz/abc",      ""},
        {"xyz/abc/x",       "xyz/abc/y",    "xyz/abc"},
        {"",                "",             ""},
        {"abc/ddd",         "xyz/efg",      ""},
    }
    testutil.TestStringOp2(t, testData, CommonSubPath)
}

func TestDirLevel(t *testing.T) {
    type rowt struct{input string; expected int}
    testData := []rowt{
        // dir          expected
        {"efg",         1},
        {"abc/xyz",     2},
        {"abc/xyz/xyz", 3},
        {"",            1},
    }
    for _, row := range testData {
        result := DirLevel(row.input)
        if result != row.expected {
            t.Error("input =", row.input, "| Expected", row.expected,
                "got", result)
        }
    }
}
