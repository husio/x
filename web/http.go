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

// JSONResp write content as JSON encoded response.
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

// JSONErr write single error as JSON encoded response.
func JSONErr(w http.ResponseWriter, errText string, code int) {
	JSONErrs(w, []string{errText}, code)
}

// JSONErrs write multiple errors as JSON encoded response.
func JSONErrs(w http.ResponseWriter, errs []string, code int) {
	resp := struct {
		Code   int
		Errors []string `json:"errors"`
	}{
		Code:   code,
		Errors: errs,
	}
	JSONResp(w, resp, code)
}

// StdJSONResp write JSON encoded, standard HTTP response text for given status
// code. Depending on status, either error or successful response format is
// used.
func StdJSONResp(w http.ResponseWriter, code int) {
	if code >= 400 {
		JSONErr(w, http.StatusText(code), code)
	} else {
		JSONResp(w, http.StatusText(code), code)
	}
}

// JSONRedirect return redirect response, but with JSON formatted body.
func JSONRedirect(w http.ResponseWriter, urlStr string, code int) {
	w.Header().Set("Location", urlStr)
	var content = struct {
		Code     int
		Location string
	}{
		Code:     code,
		Location: urlStr,
	}
	JSONResp(w, content, code)
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

func LogCall(out io.Writer, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &respwrt{code: http.StatusOK, ResponseWriter: w}
		start := time.Now()
		fn(rw, r)
		path := r.URL.String() + strings.Repeat(".", 60-len(r.URL.String()))
		fmt.Fprintf(out, "%4s %d %s %s\n", r.Method, rw.code, path, time.Now().Sub(start))
	}
}

type respwrt struct {
	code int
	http.ResponseWriter
}

func (w *respwrt) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// CheckLastModified check given request for If-Modified-Since header and if
// present, compares it with given modification time. If no modification was
// made, NotModified response is written and true returned. Otherwise
// Last-Modified header is set for the writer and false returned.
func CheckLastModified(w http.ResponseWriter, r *http.Request, modtime time.Time) bool {
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

// RedirectHandler return HandlerFunc that always redirect to given url.
func RedirectHandler(url string, code int) HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, url, code)
	}
}

// StdTextHandler return HandlerFunc that always response with text/plain
// formatted, standard for given status code text message.
func StdTextHandler(code int) HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(code)
		fmt.Fprintln(w, http.StatusText(code))
	}
}

// StdJSONHandler return HandlerFunc that always response with JSON encoded,
// standard for given status code text message.
func StdJSONHandler(code int) HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		StdJSONResp(w, code)
	}
}

func StaticHandler(root string) HandlerFunc {
	h := http.StripPrefix("/"+root, http.FileServer(http.Dir(root)))
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
}
