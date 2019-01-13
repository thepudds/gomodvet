// Package modvet is an API for gomodvet (a simple prototype of a potential future 'go mod vet' or similar).
//
// See the README at https://github.com/thepudds/gomodvet for more details.
package modvet

import (
	"fmt"
	"os/exec"
	"regexp"
)

// CheckModNeedsUpdate reports if the current 'go.mod' would be updated by
// a 'go build', 'go list', or similar command.
// Rule: gomodvet-001.
func CheckModNeedsUpdate(verbose bool) (bool, error) {

	// TODO: better way to check this that is more specific to readonly.

	// Note that 'go list -mod=readonly -m all' does not complain if an update is needed,
	// but 'go list -mod=readonly' does complain.
	out, err := exec.Command("go", "list", "-mod=readonly", "./...").CombinedOutput()
	if err != nil {
		if verbose {
			fmt.Println("gomodvet: error reported when running 'go list -mod=readonly':", string(out))
		}

		out2, err2 := exec.Command("go", "list", "./...").CombinedOutput()
		if err2 != nil {
			// error with -mod=readonly, but also without -mod=readonly, so this is likely an error
			// unrelated to whether or not an update is needed.
			fmt.Println("gomodvet: error reported when running 'go list':", string(out2))
			return false, err2
		}
		// // error with -mod=readonly, but not without -mod=readonly, so likely due to the -mod=readonly
		return true, nil
	}
	return false, nil
}

// CheckUpgrades checks if the are any upgrades for any direct and indirect dependencies.
// It returns true if upgrades are needed.
// Rule: gomodvet-002
func CheckUpgrades(verbose bool) (bool, error) {
	mods, err := ModAll()
	if err != nil {
		return false, err
	}
	flagged := false
	for _, mod := range mods {
		if verbose {
			fmt.Printf("gomodvet: checkupgrades: module %s: %+v\n", mod.Path, mod)
		}
		if mod.Update != nil {
			fmt.Println("gomodvet-002: dependencies have available updates: ", mod.Path, mod.Update.Version)
			flagged = true
		}
	}
	return flagged, nil
}

// CheckMultipleMajor checks if the current module has any dependencies with a module path has multiple major versions.
// For example, if the current module is 'foo', it reports if there is a 'bar' and 'bar/v3' as dependcies of 'foo'.
// It returns true if multiple major versions are found.
// Rule: gomodvet-003
func CheckMultipleMajor(verbose bool) (bool, error) {
	// TODO: non-regexp parsing of '/vN'?
	re := regexp.MustCompile("/v[0-9]+$")
	modPaths := make(map[string]string)
	mods, err := ModAll()
	if err != nil {
		fmt.Println("gomodvet:", err)
		return false, err
	}

	flagged := false
	for _, mod := range mods {
		if verbose {
			fmt.Printf("gomodvet: checkmultiplemajors: module %s: %+v\n", mod.Path, mod)
		}
		strippedPath := re.ReplaceAllString(mod.Path, "")
		if priorPath, ok := modPaths[strippedPath]; ok {
			fmt.Println("gomodvet-003: a module has multiple major versions in this build: ", priorPath, mod.Path)
			flagged = true
		}
		modPaths[strippedPath] = mod.Path
	}
	return flagged, nil
}
