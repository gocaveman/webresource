package webresource

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// fullPathSplit does "/a/b/c" -> "a", "b", "c".
// Absolute paths only.
func fullPathSplit(p string) []string {

	var ret []string

	p1 := path.Clean("/" + p)
	for {
		dir, base := path.Split(p1)
		p1 = strings.TrimSuffix(dir, "/")
		if base == "" {
			break
		}
		ret = append(ret, base)
	}

	// reverse
	for i, j := 0, len(ret)-1; i < j; i, j = i+1, j-1 {
		ret[i], ret[j] = ret[j], ret[i]
	}

	return ret
}

func NewFileSet(name string, requires ...Module) *FileSet {
	return &FileSet{
		name:     name,
		requires: requires,
		root: &fileEntry{
			name:    "/",
			buf:     nil,
			mode:    0755 | os.ModeDir,
			modTime: time.Now(),
			sys:     nil,
		},
	}
}

// FileSet implements Module and makes it easy to add file contents.
type FileSet struct {
	root     *fileEntry
	name     string
	requires []Module
}

func (fs *FileSet) Name() string       { return fs.name }
func (fs *FileSet) Requires() []Module { return fs.requires }

func (fs *FileSet) String() string {

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s", fs.name)

	if len(fs.requires) == 0 {
		return buf.String()
	}

	fmt.Fprintf(&buf, " -> (")

	for _, r := range fs.requires {
		fmt.Fprintf(&buf, "%s, ", r)
	}
	buf.Truncate(buf.Len() - 2)
	// if bytes.HasSuffix(buf.Bytes(), []byte(", ")) {
	// buf.Truncate(buf.Len() - 2)
	// }

	fmt.Fprintf(&buf, ")")

	return buf.String()
}

// traverse to find the entry for a path, nil if not found
func (fs *FileSet) findEntry(fullPath string) *fileEntry {
	parts := fullPathSplit(path.Clean("/" + fullPath))
	thisEntry := fs.root
	for _, part := range parts {
		e2 := thisEntry.children.entryWithName(part)
		if e2 == nil {
			return nil
		}
		thisEntry = e2
	}
	return thisEntry
}

// Mkdir creates a directory.
// It must not already exists but its parents must, will panic otherwise.
// Readdir() will return files and directories in the sequence they are created.
func (fs *FileSet) Mkdir(fullPath string, mode os.FileMode) *FileSet {

	fullPath = path.Clean("/" + fullPath)

	dir, base := path.Split(fullPath)
	dire := fs.findEntry(dir)

	if dire == nil {
		panic(fmt.Errorf("no entry found for parent directory %q", dir))
	}

	if !dire.IsDir() {
		panic(fmt.Errorf("%q is not a directory", dir))
	}

	basee := dire.children.entryWithName(base)
	if basee != nil {
		panic(fmt.Errorf("entry already exists for %q in directory %q", base, dir))
	}

	dire.children = append(dire.children,
		newFileEntry(base, nil, mode|os.ModeDir, time.Now(), nil))

	return fs
}

// WriteFile creates a file.
// It must not already exists but its parents must, will panic otherwise.
// The modTime argument is included because it is useful to indicate the original
// build time of the file.
// Readdir() will return files and directories in the sequence they are created.
func (fs *FileSet) WriteFile(fullPath string, mode os.FileMode, modTime time.Time, contents []byte) *FileSet {

	fullPath = path.Clean("/" + fullPath)

	dir, base := path.Split(fullPath)
	dire := fs.findEntry(dir)

	if dire == nil {
		panic(fmt.Errorf("no entry found for parent directory %q", dir))
	}

	if !dire.IsDir() {
		panic(fmt.Errorf("%q is not a directory", dir))
	}

	basee := dire.children.entryWithName(base)
	if basee != nil {
		panic(fmt.Errorf("entry already exists for %q in directory %q", base, dir))
	}

	dire.children = append(dire.children,
		newFileEntry(base, contents, mode, modTime, nil))

	return fs
}

// Open implements http.FileSystem.
func (fs *FileSet) Open(fullPath string) (http.File, error) {
	e := fs.findEntry(fullPath)
	if e == nil {
		return nil, os.ErrNotExist
	}
	return e.open(), nil
}

func newFileEntry(name string, b []byte, mode os.FileMode, modTime time.Time, sys interface{}) *fileEntry {

	// sanity check - name should not only be the base name component, no other parth parts
	d, n := path.Split(name)
	if d != "" || n != name {
		panic(fmt.Errorf("invalid name %q", name))
	}

	return &fileEntry{
		name:    name,
		buf:     bytes.NewBuffer(b),
		mode:    mode,
		modTime: modTime,
		sys:     sys,
	}
}

type fileEntry struct {
	buf      *bytes.Buffer // contents of the file
	name     string        // name component of the file/dir, will never contain a slash except for root
	mode     os.FileMode
	modTime  time.Time
	sys      interface{}
	children fileEntryList // for directories, the child entries
}

func (fe *fileEntry) open() *file {
	var b []byte
	if fe.buf != nil {
		b = fe.buf.Bytes()
	}
	return &file{
		fileEntry: fe,
		Reader:    bytes.NewReader(b),
		children:  fe.children,
	}
}

// make *fileEntry implement os.FileInfo so it can just return itself from Stat()

func (fe *fileEntry) Name() string       { return fe.name }
func (fe *fileEntry) Size() int64        { return int64(fe.buf.Len()) }
func (fe *fileEntry) Mode() os.FileMode  { return fe.mode }
func (fe *fileEntry) ModTime() time.Time { return fe.modTime }
func (fe *fileEntry) IsDir() bool        { return fe.mode.IsDir() }
func (fe *fileEntry) Sys() interface{}   { return fe.sys }

func (fe *fileEntry) Stat() (os.FileInfo, error) {
	return fe, nil
}

type fileEntryList []*fileEntry

func (l fileEntryList) entryWithName(name string) *fileEntry {
	for _, fe := range l {
		if fe.name == name {
			return fe
		}
	}
	return nil
}

// file implements http.File using a fileEntry
type file struct {
	*fileEntry                  // implements most of the stuff we need
	*bytes.Reader               // a reader for our specific opened instance of this fileEntry
	children      fileEntryList // needed by Readdir()
}

func (f *file) Close() error {
	_, err := f.Reader.Seek(0, io.SeekEnd)
	return err
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {

	if count <= 0 { // no limit case
		count = len(f.children)
	}

	if count > len(f.children) { // don't overflow
		count = len(f.children)
	}

	// extract what we need and remove from list to set up for next Readdir() call
	ch := f.children[:count]
	f.children = f.children[count:]

	// convert *file->os.FileInfo
	ret := make([]os.FileInfo, 0, len(ch))
	for _, c := range ch {
		ret = append(ret, c)
	}

	if len(ret) == 0 {
		return nil, io.EOF
	}

	return ret, nil
}
