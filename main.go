// gomodvet is a simple prototype of a potential future 'go mod vet' or similar.
//
// See the README at https://github.com/thepudds/gomodvet for more details.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thepudds/gomodvet/buildlist"
	"github.com/thepudds/gomodvet/vet"
)

var (
	flagConflictingRequires = flag.Bool("conflictingrequires", true, "report if there are requirements for potentially conflicting v0 versions or '+incompatible' versions for different major versions")
	flagExcludedVersion     = flag.Bool("excludedversion", true, "report if the current build is using a version excluded by a dependency")
	flagMultipleMajor       = flag.Bool("multiplemajor", true, "report if a module has multiple major versions in use")
	flagPrerelease          = flag.Bool("prerelease", true, "report if the current build is using a prerelease version (exclusive of pseudo-versions, which are reported separately)")
	flagPseudoVersion       = flag.Bool("pseudoversion", true, "report if the current build is using a pseudo-version")
	flagReplace             = flag.Bool("replace", true, "report if the main module is using any 'replace' directives")
	flagUpgrades            = flag.Bool("upgrades", true, "report if the current module has available updates for its dependencies")
	flagVerbose             = flag.Bool("v", false, "verbose: show additional information")
)

// constants for status codes for os.Exit()
const (
	Success  = 0
	OtherErr = 1
	ArgErr   = 2
)

func main() {
	os.Exit(gomodvetMain())
}

// gomodvetMain implements main(), returning a status code usable by os.Exit() and the testscript package.
// Success is status code 0.
func gomodvetMain() int {

	flag.Parse()

	status := Success
	// check we have a current go.mod
	modExists, err := buildlist.InModule()
	if err != nil {
		// TODO: stderr? stdout? For now, mostly stdout across the board.
		fmt.Println("gomodvet:", err)
		return OtherErr
	}
	if !modExists {
		fmt.Println("gomodvet: no current 'go.mod' file. please run from within a module with module-mode enabled.")
		return OtherErr
	}

	// gomodvet-001
	updateNeeded, err := vet.GoModNeedsUpdate(*flagVerbose)
	if err != nil {
		fmt.Println("gomodvet:", err)
		return OtherErr
	}
	if updateNeeded {
		// we probably should not proceed in this case, so report, then return to end our processing.
		fmt.Println("gomodvet: exiting prior to checking other rules.")
		return OtherErr
	}

	// loop over our remaining vet checks
	funcs := []struct {
		flag    *bool
		vetFunc func(bool) (bool, error)
	}{
		{flagUpgrades, vet.Upgrades},                       // gomodvet-002
		{flagMultipleMajor, vet.MultipleMajor},             // gomodvet-003
		{flagConflictingRequires, vet.ConflictingRequires}, // gomodvet-004
		{flagExcludedVersion, vet.ExcludedVersion},         // gomodvet-005
		{flagPrerelease, vet.Prerelease},                   // gomodvet-006
		{flagPseudoVersion, vet.PseudoVersion},             // gomodvet-007
		{flagReplace, vet.Replace},                         // gomodvet-008
	}

	for i := range funcs {
		if *funcs[i].flag {
			flagged, err := funcs[i].vetFunc(*flagVerbose)
			if err != nil {
				fmt.Println("gomodvet:", err)
				return OtherErr
			}
			if flagged {
				status = OtherErr
			}
		}
	}

	return status
}
