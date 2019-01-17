// Package vet is an API for gomodvet (a simple prototype of a potential future 'go mod vet' or similar).
//
// See the README at https://github.com/thepudds/gomodvet for more details.
package vet

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/rogpeppe/go-internal/semver"
	"github.com/thepudds/gomodvet/buildlist"
	"github.com/thepudds/gomodvet/modfile"
	"github.com/thepudds/gomodvet/modgraph"
)

// GoModNeedsUpdate reports if the current 'go.mod' would be updated by
// a 'go build', 'go list', or similar command.
// Rule: gomodvet-001.
func GoModNeedsUpdate(verbose bool) (bool, error) {

	// TODO: better way to check this that is more specific to readonly.
	// Probably better to check 'go' output for the specific error?

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
		fmt.Println("gomodvet-001: the current module's 'go.mod' file would be updated by a 'go build' or 'go list. Please update prior to using gomodvet.")
		return true, nil
	}
	return false, nil
}

// Upgrades reports if the are any upgrades for any direct and indirect dependencies.
// It returns true if upgrades are needed.
// Rule: gomodvet-002
func Upgrades(verbose bool) (bool, error) {
	mods, err := buildlist.ResolveUpgrades()
	if err != nil {
		return false, err
	}
	flagged := false
	for _, mod := range mods {
		if verbose {
			fmt.Printf("gomodvet: upgrades: module %s: %+v\n", mod.Path, mod)
		}
		if mod.Update != nil {
			fmt.Println("gomodvet-002: dependencies have available updates: ", mod.Path, mod.Update.Version)
			flagged = true
		}
	}
	return flagged, nil
}

// MultipleMajor reports if the current module has any dependencies with multiple major versions.
// For example, if the current module is 'foo', it reports if there is a 'bar' and 'bar/v3' as dependencies of 'foo'.
// It returns true if multiple major versions are found.
// Note that this looks for Semantic Import Version '/vN' versions, not gopkg.in versions. (Probably reasonable to not flag gopkg.in?)
// Could use SplitPathVersion from https://github.com/rogpeppe/go-internal/blob/master/module/module.go#L274
// Rule: gomodvet-003
func MultipleMajor(verbose bool) (bool, error) {
	// TODO: non-regexp parsing of '/vN'?
	re := regexp.MustCompile("/v[0-9]+$")
	// track our paths in { strippedPath: fullPath, ... } map.
	paths := make(map[string]string)
	mods, err := buildlist.Resolve()
	if err != nil {
		fmt.Println("gomodvet:", err)
		return false, err
	}

	flagged := false
	for _, mod := range mods {
		if verbose {
			fmt.Printf("gomodvet: multiplemajors: module %s: %+v\n", mod.Path, mod)
		}
		strippedPath := re.ReplaceAllString(mod.Path, "")
		if priorPath, ok := paths[strippedPath]; ok {
			fmt.Println("gomodvet-003: a module has multiple major versions in this build: ", priorPath, mod.Path)
			flagged = true
		}
		paths[strippedPath] = mod.Path
	}
	return flagged, nil
}

// ConflictingRequires reports if the current module or any dependencies have:
//    -- different v0 versions of a shared dependency.
//    -- a v0 version of a shared dependency plus a v1 version.
//    -- a vN+incompatible (N > 2) version of a shared dependency plus a v0, v1, or other vN+incompatible.
// It returns true if so.
// Rule: gomodvet-004
func ConflictingRequires(verbose bool) (bool, error) {
	// obtain the set of requires by all modules in our build (via 'go mod graph').
	// this takes into account replace directives.
	requires, err := modgraph.Requirements()
	if err != nil {
		return false, err
	}

	// track our paths and versions in { path: {version, version, ...}, ... } map.
	paths := make(map[string][]string)
	for _, require := range requires {
		f := strings.Split(require, "@")
		if len(f) != 2 {
			return false, fmt.Errorf("unexpected requirement: %s", require)
		}
		path, version := f[0], f[1]
		if !semver.IsValid(version) {
			return false, fmt.Errorf("invalid semver version: %s", require)
		}

		// Probably not needed, but might as well use the canonical semver version. That strips "+incompatible",
		// which we need to preserve. Thus, we check here for "+incompatible" and add it back if needed.
		if semver.Build(version) == "+incompatible" {
			paths[path] = append(paths[path], semver.Canonical(version)+"+incompatible")
		} else {
			paths[path] = append(paths[path], semver.Canonical(version))
		}
	}

	// for each path, loop over its versions (in semantic order) and build up a list
	// of potential conflicts.
	flagged := false
	for path, versions := range paths {
		sort.Slice(versions, func(i, j int) bool { return -1 == semver.Compare(versions[i], versions[j]) })

		if verbose {
			fmt.Printf("gomodvet: conflictingrequires: module %q has require versions: %v\n", path, versions)
		}

		priorVersion := ""
		var potentialIncompats []string
		for _, version := range versions {
			if version == priorVersion {
				continue
			}
			if isBeforeV1(version) {
				// all pre-v1 versions are potentially incompatible
				potentialIncompats = append(potentialIncompats, version)
			} else if isV1(version) && !isV1(priorVersion) {
				// the first v1 version seen is potentially incompatible with any v0, v2+incompatible, v3+incompatible, etc.
				potentialIncompats = append(potentialIncompats, version)
			} else if isV2OrHigherIncompat(version) && semver.Major(version) != semver.Major(priorVersion) {
				// the first major version v2+incompatible, v3+incompatible, etc is potentially incompatible.
				// (If two v2+incompatible versions are seen, in theory they should be compatible with each other).
				potentialIncompats = append(potentialIncompats, version)
			}
			priorVersion = version
		}
		if len(potentialIncompats) > 1 {
			// mutiple potential incompatible versions, which means they can be incompatible with each other.
			fmt.Printf("gomodvet-004: module %q was required with potentially incompatible versions: %s\n",
				path, strings.Join(potentialIncompats, ", "))
			flagged = true
		}
	}
	return flagged, nil
}

// ExcludedVersion reports if the current module or any dependencies are using a version excluded by a dependency.
// It returns true if so.
// Currently requires main module's go.mod being in a consistent state (e.g., after a 'go list' or 'go build'), such that
// the main module does not have a go.mod file using something it excludes.
// gomodvet enforces this requirement.
//
// ExcludedVersion also assumes versions in any 'go.mod' file in the build is using canonical version strings.
// The 'go' tool also enforces this when run (with some rare possible exceptions like multiple valid tags for a single commit),
// but a person could check in any given 'go.mod' file prior to letting the 'go' tool use canonical version strings. If
// that were to happen, the current ExcludedVersion could have a false negative (that is, potentially miss flagging something).
// Rule: gomodvet-005
func ExcludedVersion(verbose bool) (bool, error) {
	report := func(err error) error { return fmt.Errorf("excludedversion: %v", err) }

	// track our versions in { path: version } map.
	versions := make(map[string]string)
	mods, err := buildlist.Resolve()
	if err != nil {
		return false, report(err)
	}
	// build up our reference map
	for _, mod := range mods {
		if verbose {
			fmt.Printf("gomodvet: excludedversion: module %s: %+v\n", mod.Path, mod)
		}
		versions[mod.Path] = mod.Version
	}

	// do our check by parsing each 'go.mod' file being used,
	// and check if we are using a path/version combination excluded
	// by one of a go.mod file in our dependecies
	flagged := false
	for _, mod := range mods {
		if mod.Main {
			// here we assume the main module's 'go.mod' is in a consistent state,
			// and not using something excluded in its own 'go.mod' file. The 'go' tool
			// enforces this on a 'go build', 'go mod tidy', etc.
			continue
		}
		file, err := modfile.Parse(mod.GoMod)
		if err != nil {
			return false, report(err)
		}
		for _, exclude := range file.Exclude {
			usingVersion, ok := versions[exclude.Path]
			if !ok {
				continue
			}
			if usingVersion == exclude.Version {
				fmt.Printf("gomodvet-005: a module is using a version excluded by another module. excluded version: %s %s\n",
					exclude.Path, exclude.Version)
				flagged = true
			}
		}
	}
	return flagged, nil
}

// Prerelease reports if the current module or any dependencies are using a prerelease semver version
// (exclusive of pseudo-versions, which are also prerelease versions according to semver spec but are reported separately).
// It returns true if so.
// Rule: gomodvet-006
func Prerelease(verbose bool) (bool, error) {
	mods, err := buildlist.Resolve()
	if err != nil {
		return false, fmt.Errorf("prerelease: %v", err)
	}

	flagged := false
	for _, mod := range mods {
		if verbose {
			fmt.Printf("gomodvet: prerelease: module %s: %+v\n", mod.Path, mod)
		}
		if isPrerelease(mod.Version) {
			fmt.Printf("gomodvet-006: a module is using a prerelease version: %s %s\n",
				mod.Path, mod.Version)
			flagged = true
		}
	}
	return flagged, nil
}

// PseudoVersion reports if the current module or any dependencies are using a prerelease semver version
// (exclusive of pseudo-versions, which are also prerelease versions according to semver spec but are reported separately).
// It returns true if so.
// Rule: gomodvet-007
func PseudoVersion(verbose bool) (bool, error) {
	mods, err := buildlist.Resolve()
	if err != nil {
		return false, fmt.Errorf("pseudoversion: %v", err)
	}

	flagged := false
	for _, mod := range mods {
		if verbose {
			fmt.Printf("gomodvet: pseudoversion: module %s: %+v\n", mod.Path, mod)
		}
		if isPseudoVersion(mod.Version) {
			fmt.Printf("gomodvet-007: a module is using a pseudoversion version: %s %s\n",
				mod.Path, mod.Version)
			flagged = true
		}
	}
	return flagged, nil
}

// Replace reports if the current go.mod has 'replace' directives.
// It returns true if so.
// The parses the 'go.mod' for the main module, and hence can report
// true if the main module's 'go.mod' has ineffective replace directives.
// Part of the use case is some people never want to check in a replace directive,
// and this can be used to check that.
// Rule: gomodvet-008
func Replace(verbose bool) (bool, error) {
	mods, err := buildlist.Resolve()
	if err != nil {
		return false, fmt.Errorf("replace: %v", err)
	}

	flagged := false
	for _, mod := range mods {
		if !mod.Main {
			continue
		}
		if verbose {
			fmt.Printf("gomodvet: replacement: module %s: %+v\n", mod.Path, mod)
		}
		file, err := modfile.Parse(mod.GoMod)
		if err != nil {
			return false, fmt.Errorf("replace: %v", err)
		}
		if len(file.Replace) > 0 {
			fmt.Printf("gomodvet-008: the main module has 'replace' directives\n")
			flagged = true
		}
	}
	return flagged, nil
}

func isPseudoVersion(version string) bool {
	// regexp from cmd/go/internal/modfetch/pseudo.go
	re := regexp.MustCompile(`^v[0-9]+\.(0\.0-|\d+\.\d+-([^+]*\.)?0\.)\d{14}-[A-Za-z0-9]+(\+incompatible)?$`)
	return semver.IsValid(version) && re.MatchString(version)
}

func isPrerelease(version string) bool {
	return semver.IsValid(version) && !isPseudoVersion(version) && semver.Prerelease(version) != ""
}

// isBeforeV1 reports if a version is prio to v1.0.0, according to semver.
// v0.9.0 and v1.0.0-alpha are examples of versions before v1.0.0.
func isBeforeV1(version string) bool {
	return semver.IsValid(version) && semver.Compare(version, "v1.0.0") < 0
}

// isV1 reports if the major version is 'v1' (e.g., 'v1.2.3')
func isV1(version string) bool {
	if !semver.IsValid(version) || isBeforeV1(version) {
		return false
	}
	return semver.Major(version) == "v1"
}

// isV2OrHigherIncompat reports if version has a v2+ major version and is "+incompatible" (e.g., "2.0.0+incompatible")
func isV2OrHigherIncompat(version string) bool {
	if !semver.IsValid(version) {
		return false
	}
	major := semver.Major(version)
	// minor nuance: here we are purposefully attempting to treat v2.0.0-alpha as a "v2" release
	return major != "v0" && major != "v1" && semver.Build(version) == "+incompatible"
}

// TODO: rule to check if mod graph has any invalid semver tags?
//        I think not possible to get invalid semver tags via mod graph; I suspect would be error.
//        already get error (as error, not vet check): invalid module version "bad": version must be of the form v1.2.3
//        or maybe if any go.mod have invalid semver tags for require?
//        go mod edit -json reject 'bad' as semver tag... so no need?
//        add test? 'require foo bad' -> invalid module version "bad": unknown revision bad
