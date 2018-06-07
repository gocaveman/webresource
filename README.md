# webresource proposal/prototype
***JS/CSS/etc dependency prototype for Go***

The `webresource` package is a prototype for how dependencies for browser libraries (JS, CSS and some others) could be implemented. See (https://github.com/golang/go/issues/25781) for discussion.

The objective is to make it possible to package JS/CSS/etc libraries as Go code and express their dependencies using the Go package dependency system (with particular attention to the transition toward [semantic import versioning](https://research.swtch.com/vgo-import)).  

For example, this provides a means to "import `bootstrap.js` and thereby automatically get its dependency `jquery.js`".  Vue.js and many, many other libraries could benefit from this same approach.

To make this idea viable, it would need to be standardized.

## Usage:

your/app/main.go
```
package main

import "github.com/gocaveman/webresource"

import "github.com/gocaveman-libs/jquery"
import "github.com/gocaveman-libs/bootstrap"

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

//go:generate mkwebresource -p "github.com/gocaveman-libs/bootstrap" -r "github.com/gocaveman-libs/jquery" .
```

The library maintainer can then use `go generate` which will invoke mkwebresource (currently at `github.com/gocaveman/webresource/mkwebresource` but presumably would go somewhere in `golang.org/x`) and package the JS and/or CSS files into a .go file (`webresource-data.go` by default).  The -r option above specifies the packages this one depends on (which in turn result in import statements and cause bootstrap's Module().Requires() to return the jquery dependency.

The `webresource.Resolve()` function shown in main.go above walks the dependency tree and returns the modules in the correct order.  And `webresource.Walk()` provides an easy way to iterate over all of the files of a specific type (file extension).

The above approach will work correctly with semantic import versioning also.  (See concerns list below for caveats.)

## What Problem This Addresses:

Go web applications often need to use JavaScript, CSS (and possibly other) files as part of their front-end/in-browser functionality.  These JS and CSS files have very similar dependency management requirements to Go code itself, and yet there is currently no way for a JS or CSS library to appropriately package itself for use in a Go program.

Allowing JS and CSS libraries to present themselves to Go code as Go packages in a standardized way allows Go package management mechanism (today `go get` and various third party tools, but tomorrow `vgo`/go1.11+ and semantic import versioning) to be used to express these dependencies; and makes it easy for Go programs to depend on these libraries in a formalized way.

Making it easy for JS and CSS library vendors to drop a couple of files into their existing repository and make it usable as a Go package has high utility.


## Working Demo Server:

The easist way to see this in action is to run the demo server:
```
go get github.com/gocaveman/webresource/demoserver
go install github.com/gocaveman/webresource/demoserver
$GOPATH/bin/demoserver
```

This shows an example web application which loads bootstrap and its dependencies.

## Concerns:

Here's a roundup of the various concerns and my initial ideas on how to address them:

- We don't specify how files are combined, minfied, or served, just exactly what is in `Module`.
- The `mkwebresource` command line utility is there to make it easy to integrate into existing JS and CSS repos.  Go generate functionality can also be used to invoke existing build processes (Grunt, etc.)
- JS and CSS vendors won't adopt overnight, but a proxy repository can be made for a library and then when the original lib adopts, the proxy can be updated to just depend on the original.  Things keep working.
- This is intended for resources with very specific rules: a) must be usable in a browser (no server-side JS, no non-browser languages), b) must not reference local site URLs (e.g. no `url(image.png)`) as they cannot be relied on, c) must be applicable to an entire web page (JS and CSS are, for practical purposes "included on a page", assets like images are not and so in order to be usable must have a known URL, and become outside our scope), d) should not be directly derivable from one of the other input files (minified version, .map files, etc. - these operations should be done after, not included in the library)
- Fonts can be supported the same was JS and CSS files can, they follow the above rules.  There may be other types of resources that also follow the rules and these would be allowed.
- This means langauges requiring transpilation are transpiled to JS beforehand (TypeScript, CoffeeScript).  Same for SASS and LESS, they become CSS before we see them in a Module.
- ES6+ should normally be transpiled to ES5, BUT it is not invalid for libraries which explicitly require a newer browser to use ES6, but they must be aware they are enforcing this decision on all libraries then depend on this one.
- JS shims/polyfills should not be depended upon by libraries directly (this would result in bloat and probably duplication as multiple libraries use different polyfills to do the same thing).  Instead libraries using newer features that might need polyfill should say that in the documentation and let the final application developer decide which polyfills to manually include to achieve what browser support.
- JS libraries with various options can be expressed as each option being an individual package that depends on the core package.  (Example: Syntax highlighter with core functionality and an option/module for each syntax it supports.)
- All files must be UTF-8, we do not support other input encodings.
- The `Module` interface is not compatible with other copies of it if moved to another package.  The convention in the prototype is libraries define `ModulePROTO()`, if the `webresource` package were moved to `golang.org/x` as a sort of test run before bringing into stdlib (as was done with `context`), the convention could be `ModuleX()`, and finally when in stdlib just `Module()`.  A module can support mulitple simulaneously without conflict.
