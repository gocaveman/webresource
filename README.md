# webresource proposal
***JS/CSS/etc dependency prototype for Go***

The `webresource` package is a prototype for how dependencies for browser libraries (JS, CSS and some others) could be implemented. See (ISSUE LINK) for discussion.

The objective is to make it possible to package JS/CSS/etc libraries as Go code and express their dependencies using the Go package dependency system (with particular attention to the transition toward [semantic import versioning](https://research.swtch.com/vgo-import)).  

For example, this provides a means to "import `bootstrap.js` and thereby automatically get its dependency `jquery.js`".  Vue.js and many, many other libraries could benefit from this same approach.

To make this idea viable, it would need to be standardized.

## Usage

your/app/main.go
```
package main

import "webresource" // in prototype: github.com/gocaveman/webresource

import "github.com/jquery/jquery" // prototype proxy for jquery: github.com/gocaveman-libs/jquery 
import "github.com/twbs/bootstrap" // prototype proxy for bootstrap: github.com/gocaveman-libs/bootstrap 

func main() {

	modList := webresource.Resolve( // performs dependency resolution
		webresource.ModuleList{
			jquery.Module(), // in prototype: ...ModulePROTO()
			jslib.Module(),
		})
	// iterate overall all files in all modules in correct sequence, with dependencies
	modList.Walk(...)
}
```

github.com/twbs/bootstrap/webresource.go
```
package bootstrap

//go:generate mkwebresource -p "github.com/gocaveman-libs/bootstrap" -r "github.com/jquery/jquery" .
```

The library maintainer can then use `go generate` which will invoke mkwebresource (currently at `github.com/gocaveman/webresource/mkwebresource` but presumably would go somewhere in `golang.org/x`) and package the JS and/or CSS files into a .go file (`webresource-data.go` by default).  The -r option above specifies the packages this one depends on (which in turn result in import statements and cause bootstrap's Module().Requires() to return the jquery dependency.

The `webresource.Resolve()` function showed in main.go above walks the dependency tree and returns the modules in the correct order.  And `webresource.Walk()` provides an easy way to iterate over all of the files of a specific type (file extension).

The above approach will work correctly with semantic import versioning also.  (See concerns list below for caveats.)

## Scope

With the aim of simplicity and focusing on the core problems, the following restrictions are applied to the scope of this package:

- fdskj
- fdsa


## Motivation

... this has the interesting implication of making Go as a language and it's package management available as an effective tool for managing in-browser resources.  I'm not aware of anyone else doing this well right now.

## JS/CSS Library Maintainers, Code Generation

It is probably not applicable as a sub command directly under 'go', but following the pattern of 
`golang.org/x/tools/cmd/stringer`, it could be `golang.org/x/tools/mkwebresource` or similar.

## Migration from prototype -> golang.org/x -> stdlib

Unfortunately the `Module` interface needs to refer to itself in order to express a dependency tree, and therefore the same Module interface in multiple packages are not directly compatible.

The solution seems to be for packages to use a different naming convention to obtaining the correct types of Module.  The prototype uses `ModulePROTO()`, if the package were moved to golang.org/x as a sort of test run before bringing into stdlib (as was done with `context`), the convention could be `ModuleX()`, and finally when in stdlib just `Module()`.  This allows packages to support more than one of these at the same time, avoiding packages breaking during migration.

## Questions

Answers to questions you are likely to ask while reading this:

### How is this better than using a CDN like unpkg.com, cdnjs.com, jsdelivr.com, etc.?

(cdns don't handle dependency management and add more possible points of failure)

### Why are the files embedded in Go code?

(can't read from package dir at runtime, "single static binary")

### Why doesn't this support images or even things like template files for XYZ JS framework?

(see rules)


