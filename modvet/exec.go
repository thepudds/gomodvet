package modvet

import (
	"bufio"
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

// ModAll returns a []Module resulting from 'go list -json -u -m all'
func ModAll() ([]Module, error) {

	var result []Module
	out, err := exec.Command("go", "list", "-mod=readonly", "-json", "-u", "-m", "all").Output()

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

// GoMod represents the detailed module information in one 'go.mod' file,
// as returned by 'go mod edit -json <path/to/go.mod>'
// From: https://golang.org/cmd/go/#hdr-Edit_go_mod_from_tools_or_scripts
type GoMod struct {
	Module  GoModModule
	Require []Require
	Exclude []Module
	Replace []Replace
}

// GoModModule represents a 'module' directive in a go.mod file,
// or a module as used in other directives such as 'replace'.6
type GoModModule struct {
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
	Old GoModModule
	New GoModModule
}

// Mod returns a GoMod resulting from ''go mod edit -json <path/to/go.mod>'
func Mod(goModFilepath string) (GoMod, error) {
	var result GoMod
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

// ModExists reports if there appears to be a current 'go.mod'.
func ModExists() (bool, error) {
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

// TODO: trim these other utils? Keep for now?

// goListDeps returns a []string of dirs for all dependencies of pkg
func goListDeps(pkg string) ([]string, error) {
	out, err := exec.Command("go", "list", "-deps", "-f", "{{.Dir}}", pkg).Output()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	results := []string{}
	for scanner.Scan() {
		results = append(results, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// goListDir returns the dir for a package import path
func goListDir(pkgPath string) (string, error) {
	out, err := exec.Command("go", "list", "-f", "{{.Dir}}", pkgPath).Output()
	if err != nil {
		return "", fmt.Errorf("failed to find directory of %v: %v", pkgPath, err)
	}
	result := strings.TrimSpace(string(out))
	if strings.Contains(result, "\n") {
		return "", fmt.Errorf("multiple directory results for package %v", pkgPath)
	}
	return result, nil
}

// A maxDuration of 0 means no max time is enforced.
func execCmd(name string, args []string, maxDuration time.Duration) error {
	report := func(err error) error { return fmt.Errorf("exec %v error: %v", name, err) }

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if maxDuration == 0 {
		// invoke cmd and let it run until it returns
		err := cmd.Run()
		if err != nil {
			return report(err)
		}
		return nil
	}

	// we have a maxDuration specified.
	// start and then manually kill cmd after maxDuration if it doesn't exit on its own.
	err := cmd.Start()
	if err != nil {
		return report(err)
	}
	timer := time.AfterFunc(maxDuration, func() {
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			// os.Interrupt expected to fail in some cases (e.g., not implemented on Windows)
			_ = cmd.Process.Kill()
		}
	})
	err = cmd.Wait()
	if timer.Stop() && err != nil {
		// timer.Stop() returned true, which means our kill func never ran, so return this error
		return report(err)
	}
	return nil
}

func info(s string, args ...interface{}) {
	// TODO: stderr? stdout?
	fmt.Println("gomodvet:", fmt.Sprintf(s, args...))
}
