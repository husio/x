package entries

import (
	"bytes"
	"net/http"
	"text/template"
)

func render(w http.ResponseWriter, name string, context interface{}, code int) {
	tmpl, err := template.ParseGlob("*/templates/*.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var b bytes.Buffer
	if err := tmpl.ExecuteTemplate(&b, name, context); err != nil {
		for _, t := range tmpl.Templates() {
			println(t.Name())
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	b.WriteTo(w)
}

func renderStdResp(w http.ResponseWriter, code int) {
	render(w, "standard-response", http.StatusText(code), code)
}

var tmpl = template.Must(template.New("").Parse(`

{{define "header"}}
<!doctype html>
<html lang="en">
	<head>
		<meta charset="UTF-8">
	</head>
{{end}}


{{define "entry-list"}}
	{{template "header" .}}
	<body>
		{{range .Entries}}
			<p>
				<a href="{{.Link}}">{{.Title}}</a> {{.Created}}
			</p>
		{{end}}
	</body>
</html>
{{end}}


{{define "resouce-list"}}
	{{template "header" .}}
	<body>
		{{range .Resources}}
			<p>
				<a href="{{.Link}}">{{.Title}}</a> {{.Created}}
			</p>
		{{end}}
	</body>
</html>
{{end}}


{{define "standard-response"}}
	{{template "header" .}}
	<body>
		<div>{{.}}</div>
	</body>
</html>
{{end}}


`))
