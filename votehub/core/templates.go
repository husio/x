package core

import (
	"html/template"
	"io"
	"log"
)

func NewTemplate(rawTemplate string) (*template.Template, error) {
	t, err := template.New("").Parse(base)
	if err != nil {
		return nil, err
	}
	return t.Parse(rawTemplate)
}

func Render(t *template.Template, w io.Writer, name string, context interface{}) error {
	err := t.ExecuteTemplate(w, name, context)
	if err != nil {
		log.Printf("cannot render %q: %s", name, err)
	}
	return err
}

const base = `

{{define "header"}}
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <meta name="description" content="">
    <meta name="author" content="">
    <title>VoteHub</title>
	<link href="//maxcdn.bootstrapcdn.com/bootstrap/4.0.0-alpha.2/css/bootstrap.min.css" rel="stylesheet">
  </head>
  <body>
    <div class="container">
{{end}}


{{define "footer"}}
    </div>
  </body>
</html>
{{end}}
`
