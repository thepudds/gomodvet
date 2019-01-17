// Package buildlist is an API for gomodvet (a simple prototype of a potential future 'go mod vet' or similar).
//
// buildlist uses the context of the active module based on the current working directory to
// allow examination of the build list.
// See https://golang.org/cmd/go/#hdr-The_main_module_and_the_build_list for more on the build list.
//
// See the README at https://github.com/thepudds/gomodvet for more details on gomodvet.
package buildlist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Module represents summary Go module information, as returned by 'go list -json -m all'.
// From: https://golang.org/cmd/go/#hdr-List_packages_or_modules
// Note that a Module here is distinct from and heavierweight than a modfile.Module.
type Module struct {
	Path     string       // module path
	Version  string       // module version
	Versions []string     // available module versions (with -versions)
	Replace  *Module      // replaced by this module
	Time     *time.Time   // time version was created
	Update   *Module      // available update, if any (with -u)
	Main     bool         // is this the main module?
	Indirect bool         // is this module only an indirect dependency of main module?
	Dir      string       // directory holding files for this module, if any
	GoMod    string       // path to go.mod file for this module, if any
	Error    *ModuleError // error loading module
}

// ModuleError represents an error loading a module
type ModuleError struct {
	Err string // the error itself
}

// Resolve returns the build list in the form of []Module as returned from 'go list -json -m all'.
// In general, the "build list" is defined to be the final set of versions of modules
// providing packages to this build, including taking into account minimal version selection,
// excludes, and replaces.
// See https://golang.org/cmd/go/#hdr-The_main_module_and_the_build_list
func Resolve() ([]Module, error) {
	return resolve(false)
}

// ResolveUpgrades returns the build list (including upgrades) in the form of []Module
// as returned from 'go list -u -json -m all'.
// See Resolve for details.
func ResolveUpgrades() ([]Module, error) {
	return resolve(true)
}

func resolve(upgrades bool) ([]Module, error) {
	var result []Module
	var args []string
	if upgrades {
		args = []string{"list", "-mod=readonly", "-json", "-u", "-m", "all"}
	} else {
		args = []string{"list", "-mod=readonly", "-json", "-m", "all"}
	}
	out, err := exec.Command("go", args...).Output()

	if err != nil {
		return nil, fmt.Errorf("error invoking 'go list': %v", err)
	}

	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var m Module
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error parsing 'go list' json: %v", err)
		}
		result = append(result, m)
	}
	return result, nil
}

// InModule reports if there appears to be a current 'go.mod'.
func InModule() (bool, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return false, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" || s == os.DevNull {
		// Go 1.11 reports empty string for no 'go.mod'.
		// Go 1.12 beta (currently) reports os.DevNull for no 'go.mod'
		return false, nil
	}
	return true, nil
}

// ---------------------------------------------------
// Misc. notes on replacements
//
// Example outputs:
//
/*
$ cat go.mod
module github.com/me/hello

require rsc.io/quote v1.5.2

replace rsc.io/quote => rsc.io/quote v1.5.1

$ go list -json -m all
...
{
        "Path": "rsc.io/quote",
        "Version": "v1.5.2",
        "Replace": {
                "Path": "rsc.io/quote",
                "Version": "v1.5.1",
                "Time": "2018-02-14T00:58:40Z",
                "Dir": "...\\go\\pkg\\mod\\rsc.io\\quote@v1.5.1",
                "GoMod": "...\\go\\pkg\\mod\\cache\\download\\rsc.io\\quote\\@v\\v1.5.1.mod"
        },
        "Time": "2018-02-14T15:44:20Z",
        "Dir": "...\\go\\pkg\\mod\\rsc.io\\quote@v1.5.1",
        "GoMod": "...\\go\\pkg\\mod\\rsc.io\\quote@v1.5.1\\go.mod"
}
...

$ go mod edit -json
{
        "Module": {
                "Path": "github.com/me/hello"
        },
        "Require": [
                {
                        "Path": "rsc.io/quote",
                        "Version": "v1.5.2"
                }
        ],
        "Exclude": null,
        "Replace": [
                {
                        "Old": {
                                "Path": "rsc.io/quote"
                        },
                        "New": {
                                "Path": "rsc.io/quote",
                                "Version": "v1.5.1"
                        }
                }
        ]
}

TODO is go mod graph correct regarding replacements?

Following example seems wrong?
  "Graph prints the module requirement graph (with replacements applied) in text form"
   https://golang.org/cmd/go/#hdr-Print_module_requirement_graph

$ go mod graph
github.com/me/hello rsc.io/quote@v1.5.2
rsc.io/quote@v1.5.2 rsc.io/sampler@v1.3.0
rsc.io/sampler@v1.3.0 golang.org/x/text@v0.0.0-20170915032832-14c0d48ead0c

$ go1.12beta1 mod graph
github.com/me/hello rsc.io/quote@v1.5.2
rsc.io/quote@v1.5.2 rsc.io/sampler@v1.3.0
rsc.io/sampler@v1.3.0 golang.org/x/text@v0.0.0-20170915032832-14c0d48ead0c

$ go list -m all
github.com/me/hello
golang.org/x/text v0.0.0-20170915032832-14c0d48ead0c
rsc.io/quote v1.5.2 => rsc.io/quote v1.5.1
rsc.io/sampler v1.3.0

$ cat go.mod
module github.com/me/hello

require rsc.io/quote v1.5.2

replace rsc.io/quote => rsc.io/quote v1.5.1

$ cat hello.go
package main

import (
    "fmt"
    "rsc.io/quote"
)

func main() {
    fmt.Println(quote.Hello())
}

*/
