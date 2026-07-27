package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestVersionString(t *testing.T) {
	got := versionString("1.2.3")
	if !strings.Contains(got, "ghscanscope version v1.2.3") || !strings.Contains(got, runtime.GOOS+"/"+runtime.GOARCH) {
		t.Fatalf("unexpected version: %q", got)
	}
}
