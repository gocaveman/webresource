package webresource

import (
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

func TestFullPathSplit(t *testing.T) {

	tests := map[string][]string{
		"/":          nil,
		"//":         nil,
		"../":        nil,
		"/a":         []string{"a"},
		"/a/b":       []string{"a", "b"},
		"/a/b/":      []string{"a", "b"},
		"/a/b/../c":  []string{"a", "c"},
		"/a/b/c.ext": []string{"a", "b", "c.ext"},
		"//a/b":      []string{"a", "b"},
		"../../a/b":  []string{"a", "b"},
		"a/b":        []string{"a", "b"},
	}

	for testIn, testOut := range tests {
		v := fullPathSplit(testIn)
		if !reflect.DeepEqual(v, testOut) {
			t.Fail()
			t.Logf("failed check for %q, expected(%#v) actual(%#v)", testIn, testOut, v)
		}
	}

}

func TestFileSet(t *testing.T) {

	demoJsTime := time.Now()
	demoJsContents := `console.log("demo.js was here");`
	fset := NewFileSet("demo/pkg/import/path").
		Mkdir("/files", 0755).
		WriteFile("/files/demo.js", 0755, demoJsTime, []byte(demoJsContents))

	f, err := fset.Open("/files/demo.js")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != demoJsContents {
		t.Fatalf("wrong file contents: %q", string(b))
	}

}
