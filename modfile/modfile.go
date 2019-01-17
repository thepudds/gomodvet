// Package modfile is an API for gomodvet (a simple prototype of a potential future 'go mod vet' or similar).
// modfile parses 'go.mod' files, without requiring any additional context such as an active module.
//
// See the README at https://github.com/thepudds/gomodvet for more details.
package modfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// File represents the detailed module information in one 'go.mod' file,
// as returned by 'go mod edit -json <path/to/go.mod>'
// From: https://golang.org/cmd/go/#hdr-Edit_go_mod_from_tools_or_scripts
type File struct {
	Module  Module
	Require []Require
	Exclude []Module
	Replace []Replace
}

// Module represents a 'module' directive in a go.mod file,
// or a module as used in other directives such as 'replace'.
// Note that a Module here is distinct from and lighterweight than a buildlist.Module.
type Module struct {
	Path    string
	Version string
}

// Require represents a 'require' directive.
type Require struct {
	Path     string
	Version  string
	Indirect bool
}

// Replace represents a 'replace' directive.
type Replace struct {
	Old Module
	New Module
}

// Parse returns a GoMod resulting from 'go mod edit -json <path/to/go.mod>'
func Parse(goModFilepath string) (File, error) {
	var result File
	out, err := exec.Command("go", "mod", "edit", "-json", goModFilepath).Output()

	if err != nil {
		return result, fmt.Errorf("error invoking 'go mod edit -json': %v", err)
	}

	dec := json.NewDecoder(bytes.NewReader(out))
	if err := dec.Decode(&result); err != nil {
		return result, fmt.Errorf("error parsing'go mod edit -json': %v", err)
	}
	return result, nil
}
