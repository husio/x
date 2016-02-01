package votes

import (
	"fmt"
	"io"
)

func RenderBadge(w io.Writer, value int) error {
	_, err := fmt.Fprintf(w, badge, value)
	return err
}

const badge = `
<svg xmlns="http://www.w3.org/2000/svg" width="80" height="20">
	<g shape-rendering="crispEdges">
		<path fill="#E6D602" d="M0 0h20v20H0z"/>
		<path fill="#FCFFE0" d="M20 0h40v20H20"/>
	</g>
	<g text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="13" font-weight="bold">
		<text x="9" y="14" fill="#F8FFD6">+</text>
		<text x="40" y="14" fill="#141414">%d</text>
	</g>
</svg>
`
