package votes

import (
	"fmt"
	"io"
	"text/template"
)

func RenderBadge(w io.Writer, value int) error {
	ctx := badge{
		TextLeft:   "VOTEHUB",
		ColorLeft:  "#555",
		ColorRight: "#DBCB18",
		TextRight:  fmt.Sprintf("+%d", value),
	}
	return tmpl.ExecuteTemplate(w, "plastic", &ctx)
}

type badge struct {
	TextLeft   string
	ColorLeft  string
	TextRight  string
	ColorRight string
}

func (b *badge) WidthLeft() int {
	return len(b.TextLeft)*7 + padding
}

func (b *badge) WidthRight() int {
	return len(b.TextRight)*7 + padding
}

const padding = 18

var tmpl = template.Must(template.New("").Funcs(funcs).Parse(`
{{define "plastic"}}
	<svg xmlns="http://www.w3.org/2000/svg" width="{{sum .WidthLeft .WidthRight}}" height="18">
	  <linearGradient id="smooth" x2="0" y2="100%">
		<stop offset="0"  stop-color="#fff" stop-opacity=".7"/>
		<stop offset=".1" stop-color="#aaa" stop-opacity=".1"/>
		<stop offset=".9" stop-color="#000" stop-opacity=".3"/>
		<stop offset="1"  stop-color="#000" stop-opacity=".5"/>
	  </linearGradient>
	  <mask id="round">
		<rect width="{{sum .WidthLeft .WidthRight}}" height="18" rx="4" fill="#fff"/>
	  </mask>
	  <g mask="url(#round)">
		<rect width="{{.WidthLeft}}" height="18" fill="{{.ColorLeft}}"/>
		<rect x="{{.WidthLeft}}" width="{{.WidthRight}}" height="18" fill="{{.ColorRight}}"/>
		<rect width="{{sum .WidthLeft .WidthRight}}" height="18" fill="url(#smooth)"/>
	  </g>
	  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
		<text x="{{div .WidthLeft 2}}" y="14" fill="#010101" fill-opacity=".3">{{.TextLeft}}</text>
		<text x="{{div .WidthLeft 2}}" y="13">{{.TextLeft}}</text>
		<text x="{{div .WidthRight 2 | sum .WidthLeft}}" y="14" fill="#010101" fill-opacity=".3">{{.TextRight}}</text>
		<text x="{{div .WidthRight 2 | sum .WidthLeft}}" y="13">{{.TextRight}}</text>
	  </g>
	</svg>
{{end}}
`))

var funcs = template.FuncMap{
	"div": func(a, b int) int {
		return a / b
	},
	"sum": func(vals ...int) int {
		var total int
		for _, v := range vals {
			total += v
		}
		return total
	},
}
