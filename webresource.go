// Dependency management for web resources like JS, CSS and other files.
package webresource

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"
)

// Module interface describes a webresource module with a name (matching the Go import path),
// an http.FileSystem with it's contents, and slice of other Modules that this one requires.
// By convention the Module instance for a particular package can be obtained by calling that
// package's top level Module() function (NOTE: before full adoption this convention is different,
// other doc.)
type Module interface {
	http.FileSystem
	Name() string
	Requires() []Module
}

// Resolve walks the dependency tree for the resources provided and returns
// a ModuleList in the correct sequence according to dependency rules.
// The order of the input list is not important, the same input set will always
// result in the same output.
func Resolve(r ModuleList) ModuleList {

	var ret ModuleList

	// ensure we have stable sequence for input
	r2 := make(ModuleList, len(r))
	copy(r2, r)
	sort.Sort(r2)

	for _, m := range r2 {

		// for each module, recusively resolve it's dependencies
		mreqs := Resolve(m.Requires())

		// and add each one to our output, unless it's already there
		for _, mreq := range mreqs {
			if ret.Named(mreq.Name()) == nil {
				ret = append(ret, mreq)
			}
		}

		// now do this module
		if ret.Named(m.Name()) == nil {
			ret = append(ret, m)
		}

	}

	return ret
}

// ModuleList is a Module slice with some useful methods.
type ModuleList []Module

func (l ModuleList) Named(name string) Module {
	for _, wr := range l {
		if wr.Name() == name {
			return wr
		}
	}
	return nil
}

// Strings outputs the "%s" of each Module on its own line with the final newline trimmed.
func (l ModuleList) String() string {
	var buf bytes.Buffer
	for _, m := range l {
		fmt.Fprintf(&buf, "%s\n", m)
	}
	return strings.TrimSpace(buf.String())
}

// Less sorts by Name()
func (p ModuleList) Less(i, j int) bool { return p[i].Name() < p[j].Name() }
func (p ModuleList) Len() int           { return len(p) }
func (p ModuleList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type WalkFunc func(m Module, fullPath string, f http.File) error

// Walk will visit each file in each FileSystem contained in this Module by calling the fn function.
// This does not walk Requires().  Sequence is determined by the underlying Readdir() calls.
func (l ModuleList) Walk(ext string, fn WalkFunc) error {
	for _, m := range l {
		err := Walk(m, ext, fn)
		if err != nil {
			return err
		}
	}
	return nil
}

// Walk will visit each file in the FileSystem contained in this Module by calling the fn function.
// This does not walk Requires().  Sequence is determined by the underlying Readdir() calls.
func Walk(m Module, ext string, fn WalkFunc) error {
	return walkDir(m, "/", ext, fn)
}

func walkDir(m Module, root string, ext string, fn WalkFunc) error {

	dirf, err := m.Open(root)
	if err != nil {
		return err
	}
	fis, err := dirf.Readdir(-1)
	// close dir right after we're done read file infos to avoid unnecessary files left open for large trees
	dirf.Close()
	if err != nil {
		return err
	}

	for _, fi := range fis {

		// recurse into directory
		if fi.IsDir() {

			newRoot := path.Join(root, fi.Name())
			err := walkDir(m, newRoot, ext, fn)
			if err != nil {
				return err
			}

			continue
		}

		// for files...
		base := fi.Name()

		// skip wrong ext
		if path.Ext(base) != ext {
			continue
		}

		// open and close each file right here
		err := func() error {
			fullPath := path.Join(root, fi.Name())
			f, err := m.Open(fullPath)
			if err != nil {
				return err
			}
			defer f.Close()
			return fn(m, fullPath, f)
		}()
		if err != nil {
			return err
		}

	}

	return nil
}
