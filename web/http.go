package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"golang.org/x/net/context"
)

func JSONResp(w http.ResponseWriter, content interface{}, code int) {
	b, err := json.MarshalIndent(content, "", "\t")
	if err != nil {
		log.Printf("cannot JSON serialize response: %s", err)
		code = http.StatusInternalServerError
		b = []byte(`{"errors":["Internal Server Errror"]}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)

	const MB = 1 << (10 * 2)
	if len(b) > MB {
		log.Printf("response JSON body is huge: %d", len(b))
	}
	_, _ = w.Write(b)
}

func JSONErr(w http.ResponseWriter, errText string, code int) {
	resp := struct {
		Code   int
		Errors []string `json:"errors"`
	}{
		Code:   code,
		Errors: []string{errText},
	}
	JSONResp(w, resp, code)
}

func StdJSONResp(w http.ResponseWriter, code int) {
	JSONErr(w, http.StatusText(code), code)
}

func StdHTMLResp(w http.ResponseWriter, code int) {
	resp := struct {
		Code int
		Text string
	}{
		Code: code,
		Text: http.StatusText(code),
	}
	w.WriteHeader(code)
	render(w, "std-html-response", resp)
}

func StdHandler(code int) HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		StdHTMLResp(w, code)
	}
}

func HTMLErr(w http.ResponseWriter, errText string, code int) {
	content := struct {
		Code int
		Text string
	}{
		Code: code,
		Text: errText,
	}
	render(w, "error", content)
}

func render(w io.Writer, name string, content interface{}) {
	if err := tmpl.ExecuteTemplate(w, name, content); err != nil {
		log.Printf("cannot render %q template: %s", name, err)
	}
}

var tmpl = template.Must(template.New("").Parse(`

{{define "header"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
	<link href="/static/style.css" rel="stylesheet">
{{end}}


{{define "std-html-response"}}
	{{template "header" .}}
	</head>
	<body>
		<h1>
			{{.Text}}
			</small>{{.Code}}</small>
		</h1>
	</body>
</html>
{{end}}


{{define "error"}}
	{{template "header" .}}
	</head>
	<body>
		<h1>
			{{.Text}}
			</small>{{.Code}}</small>
		</h1>
	</body>
</html>
{{end}}

`))

type respwrt struct {
	code int
	http.ResponseWriter
}

func (w *respwrt) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func LogCall(out io.Writer, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &respwrt{code: http.StatusOK, ResponseWriter: w}
		start := time.Now()
		fn(rw, r)
		path := r.URL.String() + strings.Repeat(".", 60-len(r.URL.String()))
		fmt.Fprintf(out, "%4s %d %s %s\n", r.Method, rw.code, path, time.Now().Sub(start))
	}
}

func CheckLastModified(ctx context.Context, w http.ResponseWriter, r *http.Request, modtime time.Time) bool {
	if DevMode(ctx) {
		return false
	}
	// https://golang.org/src/net/http/fs.go#L273
	ms, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err == nil && modtime.Before(ms.Add(1*time.Second)) {
		h := w.Header()
		delete(h, "Content-Type")
		delete(h, "Content-Length")
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	w.Header().Set("Last-Modified", modtime.UTC().Format(http.TimeFormat))
	return false
}
