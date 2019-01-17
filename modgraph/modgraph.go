// Package modgraph is an API for gomodvet (a simple prototype of a potential future 'go mod vet' or similar).
// buildlist uses the context of the active module based on the current working directory to
// allow examination of the module requirements graph.
//
// See the README at https://github.com/thepudds/gomodvet for more details.
package modgraph

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// Requirements returns a []string of requirements for all dependencies in the build,
// with replacements applied. The form is module_path@version.
// This derived from the module requirement graph from 'go mod graph':
// https://golang.org/cmd/go/#hdr-Print_module_requirement_graph
func Requirements() ([]string, error) {
	out, err := exec.Command("go", "mod", "graph").Output()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	results := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("failed to parse line from 'go mod graph': %q", line)
		}
		results = append(results, fields[1])
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
