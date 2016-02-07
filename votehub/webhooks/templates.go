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
		<div class="col-md-8">
			{{if .Repositories}}
				<h2>Create webhooks for repositories</h2>
				<p>You can manage webhooks of {{.Repositories | len}} repositories.</p>

				<form action="/webhooks/create" method="POST">
					<table class="table table-sm">
					{{range .Repositories}}
						<tr>
							<td><input name="repository-{{.Name}}" type="checkbox"></td>
							<td><a href="https://github.com/{{.FullName}}/settings/hooks">{{.FullName}}</a></td>
							<td>{{.Description}}</td>
						</tr>
					{{end}}
					</table>

					<span name="toggle-selected" class="btn btn-info-outline">toggle</span>
					<button type="submit" class="btn btn-primary" disabled>create webhooks</button>
				</form>
			{{else}}
				<div class="alert alert-warning">You are not admin of any repository.</div>
			{{end}}
		</div>
		<div class="col-md-4">
				<h2>FAQ</h2>
				<p>Q: <strong>Why do I have to register a webhook?</strong></p>
				<p>A: Bla bla bla<p>
				<hr>
				<p>Q: <strong>Bla bla bla</strong></p>
				<p>A: Bla bla bla<p>
				<hr>
				<p>Q: <strong>My repository has webhook registered. Why checkbox is not selected?</strong></p>
				<p>A: Webhook detection is not yet implemented. Registering the same webhook more than once is secure.<p>
		</div>
	</div>

	<script>
$(function () {
	$('[name="toggle-selected"]').click(function () {
		$(':checkbox[name^=repository]').each(function () {
			$(this).prop('checked', !$(this).prop('checked'))
		})
	});
	$(':checkbox[name^=repository]').change(function () {
		$('[type=submit]').prop('disabled', $(':checkbox:checked').length === 0);
	});
});
	</script>

	{{template "footer" .}}
{{end}}

`))

func stdHTMLResp(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
