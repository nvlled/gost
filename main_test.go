package main

import (
	"github.com/nvlled/gost/testutil"
	"testing"
)

func TestUrl(t *testing.T) {
	testData := [][]string{
		//src  dest  expected
		{"abc/xyz/123", "abc/456", "../456"},
		{"abc/xyz", "efg", "../efg"},
		{"xyz", "efg", "efg"},
		{"xyz", "abc/efg", "abc/efg"},
		{"xyz", "abc/123/efg", "abc/123/efg"},
		{"abc/ddd", "xyz/efg", "../xyz/efg"},
		{"abc/xyz/123", "xyz/efg", "../../xyz/efg"},
		{"abc", "xyz/efg", "xyz/efg"},
		{"abc", "/xyz", "/xyz"},
		{"/abc/1234", "/xyz", "/xyz"},
		{"/abc", "xyz", "/abc/xyz"},
		{"xyz/efg/abc", "xyz/123", "../123"},
	}
	testutil.TestStringOp2(t, testData, genUrl)
}
