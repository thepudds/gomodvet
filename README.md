[![Build Status](https://travis-ci.org/thepudds/gomodvet.svg?branch=master)](https://travis-ci.org/thepudds/gomodvet)

# gomodvet
gomodvet is a simple prototype of a prototype for checking for potential module issues.

Very much WIP.

Currently three rules:

```
gomodvet-001: the current module's 'go.mod' file would be updated by a 'go build' or 'go list.

gomodvet-002: dependencies have available updates.

gomodvet-003: a module has multiple major versions in this build:  
github.com/thepudds/example-package-b github.com/thepudds/example-package-b/v3
```

Usage:

```
Usage of gomodvet:

  -checkmultiplemajor
        report if a module has multiple major versions in this build (default true)

  -checkupgrades
        report if the current module has available updates for its dependencies (default true)

  -v    verbose: show additional information
```
