package tmpl

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"time"
)

var (
	// XXX race conditions when in debug mode
	tmpl      = template.New("").Funcs(funcs)
	tmplGlob  string
	tmplCache bool
)

func Render(w io.Writer, name string, context interface{}) {
	if tmpl == nil {
		log.Printf("cannot render %q: templates not loaded", name)
		return
	}
	if err := tmpl.ExecuteTemplate(w, name, context); err != nil {
		log.Printf("cannot render %q: %s", name, err)
	}

	if !tmplCache {
		LoadTemplates(tmplGlob, tmplCache)
	}
}

// LoadTemplates parse and load template files matching given glob. Function is
// not thread safe and must be called only once during application
// initialization phase.
func LoadTemplates(glob string, cache bool) error {
	tmplGlob = glob
	tmplCache = cache

	t, err := tmpl.ParseGlob(glob)
	if err != nil {
		return err
	}
	tmpl = t

	return nil
}

var funcs = template.FuncMap{
	"timesince": Timesince,
}

func Timesince(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	delta := time.Now().Sub(t)
	switch {
	case delta >= year:
		if n := int(delta / year); n > 1 {
			return fmt.Sprintf("%d years ago", n)
		}
		return "1 year ago"
	case delta >= month:
		if n := int(delta / month); n > 1 {
			return fmt.Sprintf("%d months ago", n)
		}
		return "1 month ago"
	case delta >= 24*time.Hour:
		if n := int(delta / day); n > 1 {
			return fmt.Sprintf("%d days ago", n)
		}
		return "1 day ago"
	case delta >= time.Hour:
		if n := int(delta / time.Hour); n > 1 {
			return fmt.Sprintf("%d hours ago", n)
		}
		return "1 hour ago"
	case delta >= time.Minute:
		if n := int(delta / time.Minute); n > 1 {
			return fmt.Sprintf("%d minutes ago", n)
		}
		return "1 minute ago"
	case delta > 3*time.Second:
		return fmt.Sprintf("%d seconds ago", int(delta/time.Second))
	default:
		return "now"
	}
}

const (
	// more or less
	year  = 356 * 24 * time.Hour
	month = 30 * 24 * time.Hour
	day   = 24 * time.Hour
)
