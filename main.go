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
	flagCheckMultipleMajor = flag.Bool("checkmultiplemajor", false, "")
	flagCheckUpgrades      = flag.Bool("checkupgrades", false, "")
	flagVerbose            = flag.Bool("v", false, "")
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
			// TODO: our first-pass set of tests currently assume we stop on failure.
			// we could in theory keep going, but at least for now, exit.
			fmt.Println("gomodvet: exiting prior to checking other rules.")
			return OtherErr
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
			return OtherErr
		}
	}

	return Success
}
