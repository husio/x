package votes

import (
	"html/template"
	"net/http"

	"github.com/husio/x/votehub/core"
)

var tmpl = template.Must(core.NewTemplate(`

{{define "click-upvote-login"}}
	{{template "header" .}}

	<div class="row">
		<div class="col-md-4 col-md-offset-4 text-center">
			<h1>Votehub</h1>
			<p>
				Authenticate first.
			</p>
			<p>
				<a href="/login?next={{.NextURL}}" class="btn btn-success">Login with Github</a>
			</p>
		</div>
	</div>

	{{template "footer" .}}
{{end}}

`))

func stdHTMLResp(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
