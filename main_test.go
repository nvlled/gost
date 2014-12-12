package main

import (
	"github.com/nvlled/gost/testutil"
	"testing"
)

func TestUrl(t *testing.T) {
	testData := [][]string{
		{"/abc", "xyz", "xyz"},
		{"/abc", "xyz/123", "xyz/123"},
		{"/abc", "xyz/123", "xyz/123"},

		{"/abc", "/xyz/123", "xyz/123"},
		{"/abc/efg", "/xyz/123", "../xyz/123"},
		{"/abc/efg", "/abc/123", "123"},
		{"/", "/xyz", "xyz"},
		{"/", "/", "."},
		{"/", "/xxx", "xxx"},
		{"/abc", "/xxx", "xxx"},
		{"/abc", "/", "."},
		{"/aaa/abc", "/aaa", "../aaa"},
		{"/aaa/abc", "/aaa/ooo", "ooo"},
		{"/aaa/abc", "/ccc/bbb", "../ccc/bbb"},
	}
	testutil.TestStringOp2(t, testData, relativizePath)
}
