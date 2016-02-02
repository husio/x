package webhooks

import (
	"html/template"
	"net/http"

	"github.com/husio/x/votehub/core"
)

var tmpl = template.Must(core.NewTemplate(`

{{define "webhook-list"}}
	{{template "header" .}}

	<div class="row">
		<div class="col-md-12">
			<h1>Repositories</h1>
			<form action="/webhooks/create" method="POST">
				{{range .Repositories}}
					<div>
						<input name="repository-{{.Name}}" type="checkbox">
						<a href="{{.URL}}">{{.FullName}}</a>
						{{if .Description}}<small>{{.Description}}</small>{{end}}
					</div>
				{{end}}

				<button type="submit" class="btn btn-primary">Create webhooks</button>
			</form>
		</div>
	</div>

	{{template "footer" .}}
{{end}}

`))

func stdHTMLResp(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
