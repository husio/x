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

	<div class="container">
		<div class="row">
			<div class="col-md-4">
				<h2>FAQ</h2>
				<p>Q: <strong>What is votehub?</strong></p>
				<p>A: It's <strong>+1</strong> for your github project pages.<p>
				<p><a href="https://github.com/isaacs/github/issues/9">Explicit +1</a> is github's long missing functionality. Votehub provides workaround solution, couting votes per project and embeding results directly on github pages.</p>
				<hr>
				<p>Q: <strong>How does it work?</strong></p>
				<p>A: Votehub allows to embed link to SVG banners with number of votes. Because banner is also a link, clicking on adds your vote.</p>
				<hr>
				<p>Q: <strong>Do I need to create an account?</strong></p>
				<p>A: No. You must login using your github account. Depending on what you want to do, different scopes might be required.</p>
				<hr>
				<p>Q: <strong>How can add use it in my projects?</strong></p>
				<p>A: To add <em>+1</em> banners to your repository, you must login with your github account and <a href="/webhooks/create">create webhooks</a> for relevant projects. Add <code>&#123;&#123;hubtag&#125;&#125;</code> string whenever you want to insert new counter.</p>
				<hr>
				<p>Q: <strong>Who's behind it?</strong></p>
				<p>A: Just <a href="//github.com/husio">me</a>.</p>
			</div>
			<div class="col-md-4">
				<h2>Recent votes</h2>
				{{if .Votes}}
					{{range .Votes}}
						<p>
							<a href="{{.Counter.URL}}">{{.Counter.Description}}</a> {{.Vote.Created}}
							<img src="/v/{{.CounterID}}/banner.svg">
						<p>
					{{end}}
				{{else}}
					<div class="alert alert-info">
						You have not voted on anything yet.
					</div>
				{{end}}
			</div>
			<div class="col-md-4">
				<h2>Recent counters</h2>
				{{if .Counters}}
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
				{{else}}
					<div class="alert alert-info">
						You do not have any counters registered.
						<a href="/webhooks/create">Create webhook</a> for your github repository.
					</div>
				{{end}}
			</div>
		</div>
	</div>

	{{template "footer" .}}
{{end}}

`))

func stdHTMLResp(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
