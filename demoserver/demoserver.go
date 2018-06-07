// Demo of webresource - run a server with bootstrap and requirements.
// Performs CSS and JS concatenation, gzipped response, and proper cache headers,
// as well as minification (using third party package).
package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gocaveman-libs/bootstrap"
	"github.com/gocaveman/webresource"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/js"
)

const maxAgeStr = "31536000" // 1 year

func main() {

	httpListen := flag.String("http", ":8080", "Host:port to listen for http server")
	flag.Parse()

	moduleList := webresource.Resolve(webresource.ModuleList{
		bootstrap.ModulePROTO(),
		// add more modules here
	})

	min := minify.New()
	min.AddFunc("text/css", css.Minify)
	min.AddFunc("application/javascript", js.Minify)

	var err error

	// concatenate CSS, record most recent time
	var cssBuf bytes.Buffer
	var cssTime time.Time
	err = moduleList.Walk(".css", func(m webresource.Module, fullPath string, f http.File) error {
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		st, err := f.Stat()
		if err != nil {
			return err
		}
		if st.ModTime().After(cssTime) {
			cssTime = st.ModTime()
		}
		fmt.Fprintf(&cssBuf, "%s\n", b)
		return nil
	})
	cssTimeStr := fmt.Sprintf("%d", cssTime.Unix())
	var cssBufMin bytes.Buffer
	min.Minify("text/css", &cssBufMin, &cssBuf)

	// concatenate JS, record most recent time
	var jsBuf bytes.Buffer
	var jsTime time.Time
	err = moduleList.Walk(".js", func(m webresource.Module, fullPath string, f http.File) error {
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		st, err := f.Stat()
		if err != nil {
			return err
		}
		if st.ModTime().After(jsTime) {
			jsTime = st.ModTime()
		}
		fmt.Fprintf(&jsBuf, "%s\n", b)
		return nil
	})
	jsTimeStr := fmt.Sprintf("%d", jsTime.Unix())
	var jsBufMin bytes.Buffer
	min.Minify("application/javascript", &jsBufMin, &jsBuf)

	http.HandleFunc("/combined.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		if r.URL.Query().Get("_") == cssTimeStr {
			w.Header().Set("Cache-Control", "max-age="+maxAgeStr)
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		gw := gzipW(w, r)
		if gw != nil {
			w = gw
			defer gw.Close()
		}
		http.ServeContent(w, r, "/combined.css", cssTime, bytes.NewReader(cssBufMin.Bytes()))
	})

	http.HandleFunc("/combined.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		if r.URL.Query().Get("_") == jsTimeStr {
			w.Header().Set("Cache-Control", "max-age="+maxAgeStr)
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		gw := gzipW(w, r)
		if gw != nil {
			w = gw
			defer gw.Close()
		}
		http.ServeContent(w, r, "/combined.js", jsTime, bytes.NewReader(jsBufMin.Bytes()))
	})

	hometmpl, err := template.New("_home_").Parse(`<!DOCTYPE html>
<html>
<head>
	<title>demoserver - webresource server example</title>
	<link rel="stylesheet" type="text/css" href="/combined.css?_={{.cssTimeStr}}">
</head>
<body>
<div class="container">
	<h1>Demo Server Page</h1>
	<p>A simple example of using <code data-toggle="tooltip" title="" data-original-title="The webresource package is a prototype for JS and CSS dependencies in Go">webresource</code> to make a webserver.</p>
</div>
<script type="text/javascript" src="/combined.js?_={{.jsTimeStr}}"></script>
<script>
$(function () {
	$('[data-toggle="tooltip"]').tooltip(); // bootstrap made me do it: https://getbootstrap.com/docs/4.0/components/tooltips/
})
</script>
</body>
</html>
`)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		gw := gzipW(w, r)
		if gw != nil {
			w = gw
			defer gw.Close()
		}
		err := hometmpl.Execute(w, map[string]string{
			"cssTimeStr": cssTimeStr,
			"jsTimeStr":  jsTimeStr,
		})
		if err != nil {
			log.Printf("Error executing page template: %v", err)
		}
	})

	log.Printf("Listening for HTTP at %s", *httpListen)
	log.Fatal(http.ListenAndServe(*httpListen, nil))

}

// returns a gzipResponseWriter unless not supported then returns nil;
// caller is responsible for calling Close()
func gzipW(w http.ResponseWriter, r *http.Request) *gzipResponseWriter {
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w, Closer: gz}
		return &gzw
	}
	return nil
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	io.Closer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}
