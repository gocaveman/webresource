package webresource

import (
	"bytes"
	"compress/gzip"
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

	demoJsGzTime := time.Now()
	demoJsGzContents := `console.log("demogz.js was here");`
	var demoJsGzBuf bytes.Buffer
	gw := gzip.NewWriter(&demoJsGzBuf)
	_, err := gw.Write([]byte(demoJsGzContents))
	if err != nil {
		t.Fatal(err)
	}
	gw.Close()

	demoJsTime := time.Now()
	demoJsContents := `console.log("demo.js was here");`
	fset := NewFileSet("demo/pkg/import/path").
		Mkdir("/files", 0755).
		WriteFile("/files/demo.js", 0755, demoJsTime, []byte(demoJsContents)).
		WriteGzipFile("/files/demogz.js", 0755, demoJsGzTime, demoJsGzBuf.Bytes())

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

	f, err = fset.Open("/files/demogz.js")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err = ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != demoJsGzContents {
		t.Fatalf("wrong file contents: %q", string(b))
	}

	// test MkdirAll
	fset.MkdirAll("/files", 0755)
	fset.MkdirAll("/files/test2", 0755)
	df, err := fset.Open("/files/test2")
	if err != nil {
		t.Fatal(err)
	}
	defer df.Close()
	dfi, err := df.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !dfi.IsDir() {
		t.Fatalf("directory not IsDir(), should not be possible")
	}

}
