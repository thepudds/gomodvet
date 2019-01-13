// gomodvet is a simple prototype of a potential future 'go mod vet' or similar.
//
// See the README at https://github.com/thepudds/gomodvet for more details.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thepudds/gomodvet/modvet"
)

var (
	flagCheckMultipleMajor = flag.Bool("checkmultiplemajor", true, "report if a module has multiple major versions in this build")
	flagCheckUpgrades      = flag.Bool("checkupgrades", true, "report if the current module has available updates for its dependencies")
	flagVerbose            = flag.Bool("v", false, "verbose: show additional information")
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
	modExists, err := modvet.ModExists()
	if err != nil {
		// TODO: stderr? stdout? For now, mostly stdout across the board.
		fmt.Println("gomodvet:", err)
		return OtherErr
	}
	if !modExists {
		fmt.Println("gomodvet: no current 'go.mod' file. please run from within a module.")
		return OtherErr
	}

	// gomodvet-001
	updateNeeded, err := modvet.CheckModNeedsUpdate(*flagVerbose)
	if err != nil {
		fmt.Println("gomodvet:", err)
		return OtherErr
	}
	if updateNeeded {
		// we probably should not proceed in this case, so report, then return to end our processing.
		fmt.Println("gomodvet-001: the current module's 'go.mod' file would be updated by a 'go build' or 'go list. Please update prior to using gomodvet.")
		fmt.Println("gomodvet: exiting prior to checking other rules.")
		return OtherErr
	}

	// gomodvet-002
	if *flagCheckUpgrades {
		flagged, err := modvet.CheckUpgrades(*flagVerbose)
		if err != nil {
			fmt.Println("gomodvet:", err)
			return OtherErr
		}
		if flagged {
			status = OtherErr
		}
	}

	// gomodvet-003
	if *flagCheckMultipleMajor {
		flagged, err := modvet.CheckMultipleMajor(*flagVerbose)
		if err != nil {
			fmt.Println("gomodvet:", err)
			return OtherErr
		}
		if flagged {
			status = OtherErr
		}
	}

	return status
}
