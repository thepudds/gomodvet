package vet

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// TODO: trim these other utils? Keep for now?

// goListDepDirs returns a []string of dirs for all dependencies of pkg
func goListDepDirs(pkg string) ([]string, error) {
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
