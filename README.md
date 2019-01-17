[![Build Status](https://travis-ci.org/thepudds/gomodvet.svg?branch=master)](https://travis-ci.org/thepudds/gomodvet)

# gomodvet
gomodvet is a simple prototype. It is an experimental tool that checks for potential modules-related issues.

WIP.

### Rules

There are currently 8 rules:

* `gomodvet-001: the current module's go.mod file would be updated by 'go build'`
* `gomodvet-002: a module has multiple major versions in this build`
* `gomodvet-003: module "foo" was required with potentially incompatible versions: v0.9.0, v1.0.0`
* `gomodvet-004: using a version excluded by another module: github.com/go-chi/chi@v1.0.1`
* `gomodvet-005: using a prerelease version: github.com/go-chi/chi@v4.0.0-rc2`
* `gomodvet-006: using a pseudoversion version: github.com/go-chi/chi@v0.0.0-20151106203253-e413833c12f1`
* `gomodvet-007: dependencies have available updates`
* `gomodvet-008: the current module uses 'replace' directives`

Most of those are not strictly speaking "problems" in all cases, but most of those are at least
notable situations. (For example, a module with multiple major versions in a build might be a conscious
choice, or might be because someone is doing `import "foo/v3"` in one spot and accidentally 
doing `import "foo"` in another spot).

### Usage

Example invocation that checks all but two rules: `gomodvet -upgrades=false -pseudoversion=false`

```
Usage of gomodvet:

  -conflictingrequires
        report if there are requirements for potentially conflicting v0 versions or 
        '+incompatible' versions for different major versions (default true)
  
  -excludedversion
        report if the current build is using a version excluded by a dependency (default true)
  
  -multiplemajor
        report if a module has multiple major versions in use (default true)
  
  -prerelease
        report if the current build is using a prerelease version (exclusive of pseudo-versions,
        which are reported separately) (default true)
  
  -pseudoversion
        report if the current build is using a pseudo-version (default true)
  
  -replace
        report if the main module is using any 'replace' directives (default true)
  
  -upgrades
        report if the current module has available updates for its dependencies (default true)
  
  -v    verbose: show additional information
```
