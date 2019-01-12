package main

import (
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(gomodvetTestingMain{m}, map[string]func() int{
		"gomodvet": gomodvetMain,
	}))
}

type gomodvetTestingMain struct {
	m *testing.M
}

func (m gomodvetTestingMain) Run() int {
	// could do additional setup here if needed (e.g., check or set env vars, start a Go proxy server, etc.)
	return m.m.Run()
}

func TestScripts(t *testing.T) {
	p := testscript.Params{Dir: "testscripts"}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
}
