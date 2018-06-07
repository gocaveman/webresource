package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const moduleFuncName = "ModulePROTO" // prototype
// const moduleFuncName = "ModuleX" // golang.org/x
// const moduleFuncName = "Module" // go stdlib

func main() {

	importName := flag.String("p", "", "Full package import path, required, include semver major number if applicable (e.g. \"pkg/v2\")")
	outputFile := flag.String("o", "./webresource-data.go", "Output file name")
	filterExpr := flag.String("e", "\\.(js|css)$", "Filter file paths using regular expression")
	requires := flag.String("r", "", "List of full package import paths to require for this module, comma separated, empty means no requires")
	recursive := flag.Bool("R", false, "Enable recursion when scanning the directory, reproduces subdir tree in output")
	module := flag.Bool("m", true, "Set to 0 to disable the public Module() function definition, in case you want to make your own")
	flag.Parse()

	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "You must provide exactly one argument of the input directory, e.g.: mkwebresource .\n")
		os.Exit(1)
	}
	inputDir := args[0]

	*importName = strings.TrimSpace(*importName)
	if *importName == "" {
		fmt.Fprintf(os.Stderr, "You must provide an import name with -p\n")
		os.Exit(1)
	}

	// figure out package name, stripping off major semver if present
	importNameShort := importLocalName(trimMajorSemver(*importName))

	filter, err := regexp.Compile(*filterExpr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Bad filter regexp %q: %v\n", *filterExpr, err)
		os.Exit(1)
	}

	var inputFilePaths []string

	if !*recursive {
		inputDirFile, err := os.Open(inputDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input directory %q: %v\n", inputDir, err)
			os.Exit(1)
		}
		defer inputDirFile.Close()
		fis, err := inputDirFile.Readdir(-1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling readdir on input directory %q: %v\n", inputDir, err)
			os.Exit(1)
		}
		for _, fi := range fis {
			// skip dirs
			if fi.IsDir() {
				continue
			}
			// skip files that don't match filter
			if !filter.MatchString(fi.Name()) {
				continue
			}
			inputFilePaths = append(inputFilePaths, path.Clean("/"+fi.Name()))
		}
	} else {
		err := filepath.Walk(inputDir, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasPrefix(p, inputDir) {
				p = strings.TrimPrefix(p, inputDir)
			}
			if !filter.MatchString(p) {
				return nil
			}
			inputFilePaths = append(inputFilePaths, path.Clean("/"+p))
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling readdir on input directory %q: %v\n", inputDir, err)
			os.Exit(1)
		}
	}

	// log.Printf("inputFilePaths: %+v", inputFilePaths)

	var srcbuf bytes.Buffer
	fmt.Fprintf(&srcbuf, `package %s`+"\n", importNameShort)
	fmt.Fprintf(&srcbuf, "\n")

	fmt.Fprintf(&srcbuf, `import "time"`+"\n")
	fmt.Fprintf(&srcbuf, "\n")

	fmt.Fprintf(&srcbuf, `import "github.com/gocaveman/webresource" // FIXME: prototype import path for now`+"\n")
	fmt.Fprintf(&srcbuf, "\n")

	var requireList []string
	if *requires != "" {
		requireList = strings.Split(*requires, ",")
	}

	// require imports
	if len(requireList) > 0 {
		for _, r := range requireList {
			fmt.Fprintf(&srcbuf, `import %q`+"\n", r)
		}
		fmt.Fprintf(&srcbuf, "\n")
	}

	// output module stuff at the top if requested
	if *module {
		fmt.Fprintf(&srcbuf, `func %s() webresource.Module {`+"\n", moduleFuncName)
		fmt.Fprintf(&srcbuf, `fs := webresource.NewFileSet(%q, requires()...)`+"\n", *importName)
		fmt.Fprintf(&srcbuf, `addFiles(fs)`+"\n")
		fmt.Fprintf(&srcbuf, `return fs`+"\n")
		fmt.Fprintf(&srcbuf, `}`+"\n")
		fmt.Fprintf(&srcbuf, "\n")
	}

	// `requires` func
	if len(requireList) > 0 {
		fmt.Fprintf(&srcbuf, `func requires() []webresource.Module {`+"\n")
		fmt.Fprintf(&srcbuf, `return []webresource.Module{`+"\n")
		for _, r := range requireList {
			fmt.Fprintf(&srcbuf, `%s.%s(),`+"\n", importLocalName(trimMajorSemver(r)), moduleFuncName)
		}
		fmt.Fprintf(&srcbuf, `}`+"\n")
		fmt.Fprintf(&srcbuf, `}`+"\n")
	} else {
		fmt.Fprintf(&srcbuf, `func requires() []webresource.Module { return nil }`+"\n")
	}
	fmt.Fprintf(&srcbuf, "\n")

	fmt.Fprintf(&srcbuf, `func addFiles(fs *webresource.FileSet) {`+"\n")
	for _, file := range inputFilePaths {
		err := addFile(&srcbuf, inputDir, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding file %q: %v\n", file, err)
			os.Exit(1)
		}
	}
	fmt.Fprintf(&srcbuf, `}`+"\n")
	fmt.Fprintf(&srcbuf, "\n")

	srcb := srcbuf.Bytes()
	srcbfmt, err := format.Source(srcb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during gofmt: %v\n", err)
		fmt.Fprintf(os.Stderr, "Input source:\n%s\n", srcb)
		os.Exit(1)
	}

	err = ioutil.WriteFile(*outputFile, srcbfmt, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file %q: %v\n", *outputFile, err)
		os.Exit(1)
	}

}

func addFile(w io.Writer, dir string, name string) error {

	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write(b)
	if err != nil {
		return err
	}
	err = gw.Close()
	if err != nil {
		return err
	}

	nameDir, _ := path.Split(path.Clean("/" + name))
	if nameDir != "" && nameDir != "/" { // mkdirall if not root dir
		fmt.Fprintf(w, `fs = fs.MkdirAll(%q, 0755)`+"\n", nameDir)
	}
	fmt.Fprintf(w, `// compressed size: %d`+"\n", buf.Len())
	fmt.Fprintf(w, `fs = fs.WriteGzipFile(%q, 0644, time.Unix(%d, 0), []byte(%q))`+"\n", name, fi.ModTime().Unix(), buf.String())

	return nil
}

func trimMajorSemver(p string) string {
	subm := regexp.MustCompile(`(.*)/v[0-9]+$`).FindStringSubmatch(p)
	if len(subm) > 1 {
		return subm[1]
	}
	return p
}

func importLocalName(fullImportPath string) string {
	parts := strings.Split(fullImportPath, "/")
	ret := parts[len(parts)-1]
	ret = strings.NewReplacer(
		".", "",
		"-", "",
	).Replace(ret)
	return ret
}
