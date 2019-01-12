[![Build Status](https://travis-ci.org/thepudds/gomodvet.svg?branch=master)](https://travis-ci.org/thepudds/gomodvet)

# gomodvet
gomodvet is a simple prototype of a prototype for checking for potential module issues.

Very much WIP.

Currently three rules:

```
gomodvet-001: the current module's 'go.mod' file would be updated by a 'go build' or 'go list. Please update prior to using gomodvet.
gomodvet-002: the current module has available updates:  github.com/thepudds/example-package-b/v3
gomodvet-003: a module has multiple major versions in this build:  github.com/thepudds/example-package-b github.com/thepudds/example-package-b/v3
```

Usage:

```
Usage of gomodvet:
  -checkmultiplemajor

  -checkupgrades

  -v
```
