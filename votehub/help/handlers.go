package help

import (
	"html/template"
	"log"
	"net/http"

	"github.com/husio/x/auth"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/votehub/core"
	"github.com/husio/x/votehub/votes"

	"golang.org/x/net/context"
)

func HandleHelp(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := pg.DB(ctx)
	account, ok := auth.Authenticated(db, r)

	var counters []*votes.Counter
	if ok {
		res, err := votes.CountersByOwner(db, account.AccountID, 20, 0)
		if err != nil {
			log.Printf("cannot fetch entities for %d: %s", account.AccountID, err)
		} else {
			counters = res
		}
	}

	context := struct {
		Account  *auth.Account
		Counters []*votes.Counter
	}{
		Account:  account,
		Counters: counters,
	}
	core.Render(tmpl, w, "welcome", context)
}

var tmpl = template.Must(core.NewTemplate(`

{{define "welcome"}}
	{{template "header" .}}

	<div class="row">
		<div class="col-md-12">

			{{if .Counters}}
				<h3>Latest entities</h3>
				{{range .Counters}}
					<div>
						{{.Value}}) <a href="{{.URL}}">{{.URL}}</a>
						<small>{{.Description}}</small>
					</div>
				{{end}}
			{{else}}
				No counters
			{{end}}
		</div>
	</div>

	{{template "footer" .}}
{{end}}

`))
