package webresource

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func TestResolve(t *testing.T) {

	assert := func(ml ModuleList, result string) {
		mlr := Resolve(ml)
		mlrs := mlr.String()
		if mlrs != result {
			t.Fail()
			t.Logf("assert failed on module list\n-- In:\n%s\n-- Expected Output:\n%s\n-- Actual Output:\n%s\n", ml, result, mlrs)
		}
	}

	{
		a := NewFileSet("a").WriteFile("/a.js", 0644, time.Now(), []byte(`/* a.js */`))
		b := NewFileSet("b", a).WriteFile("/b.js", 0644, time.Now(), []byte(`/* b.js */`))
		c := NewFileSet("c", b).WriteFile("/c.js", 0644, time.Now(), []byte(`/* c.js */`))

		ml := ModuleList{c}

		assert(ml, `a
b -> (a)
c -> (b -> (a))`)
	}

	{
		a := NewFileSet("a").WriteFile("/a.js", 0644, time.Now(), []byte(`/* a.js */`))
		b := NewFileSet("b", a).WriteFile("/b.js", 0644, time.Now(), []byte(`/* b.js */`))
		c := NewFileSet("c", a).WriteFile("/c.js", 0644, time.Now(), []byte(`/* c.js */`))

		ml := ModuleList{c, b}

		assert(ml, `a
b -> (a)
c -> (a)`)
	}

	{
		a := NewFileSet("a").WriteFile("/a.js", 0644, time.Now(), []byte(`/* a.js */`))
		b := NewFileSet("b", a).WriteFile("/b.js", 0644, time.Now(), []byte(`/* b.js */`))
		c := NewFileSet("c", a).WriteFile("/c.js", 0644, time.Now(), []byte(`/* c.js */`))
		d := NewFileSet("d", b, c).WriteFile("/d.js", 0644, time.Now(), []byte(`/* d.js */`))

		ml := ModuleList{d}

		assert(ml, `a
b -> (a)
c -> (a)
d -> (b -> (a), c -> (a))`)
	}

}

func TestWalk(t *testing.T) {

	{
		a := NewFileSet("a").WriteFile("/a.js", 0644, time.Now(), []byte(`/* a.js */`))
		b := NewFileSet("b", a).WriteFile("/b.js", 0644, time.Now(), []byte(`/* b.js */`))
		c := NewFileSet("c", a).WriteFile("/c.js", 0644, time.Now(), []byte(`/* c.js */`))
		d := NewFileSet("d", b, c).
			Mkdir("/subdir", 0755).
			WriteFile("/subdir/d1.js", 0644, time.Now(), []byte(`/* d1.js */`)).
			WriteFile("/subdir/d2.js", 0644, time.Now(), []byte(`/* d2.js */`))

		ml := ModuleList{d}

		mlr := Resolve(ml)
		var buf bytes.Buffer
		err := mlr.Walk(".js", func(m Module, fullPath string, f http.File) error {
			b, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			fmt.Fprintf(&buf, "%s", b)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if buf.String() != `/* a.js *//* b.js *//* c.js *//* d1.js *//* d2.js */` {
			t.Fatalf("expected result: %s", buf.String())
		}

	}

}
