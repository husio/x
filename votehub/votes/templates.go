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


{{define "counters-list"}}
	{{template "header" .}}

	<div class="row">
		<div class="col-md-12">
			<h1>Counters</h1>
			{{range .Counters}}
				<div class="row">
					<div class="col-md-12">
						<img src="/v/{{.CounterID}}/banner.svg">
						<p>
							{{.Description}}
							<small>{{.Created}}</small>
						</p>
						<a href="{{.URL}}">{{.URL}}</a>
					</div>
				</div>
			{{end}}
		</div>
	</div>

	{{template "footer" .}}
{{end}}

`))

func stdHTMLResp(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
