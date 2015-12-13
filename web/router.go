package web

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/context"
)

type Routes []Route

type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

type Route struct {
	Methods string
	Path    string
	Func    HandlerFunc
}

func NewRouter(prefix string, routes Routes) *Router {
	handlers := make(map[string][]handler)
	builder := regexp.MustCompile("{.*?}")
	// ReplaceAllString

	for _, r := range routes {
		var names []string
		raw := builder.ReplaceAllStringFunc(prefix+r.Path, func(s string) string {
			s = s[1 : len(s)-1]
			// every {<name>} can be optionally contain separate regexp
			// definition using notation {<name>:<regexp>}
			chunks := strings.SplitN(s, ":", 2)
			if len(chunks) == 1 {
				names = append(names, s)
				return `([^/]+)`
			}
			names = append(names, chunks[0])
			return `(` + chunks[1] + `)`
		})
		// replace {} with regular expressions syntax
		rx, err := regexp.Compile(`^` + raw + `$`)
		if err != nil {
			panic(fmt.Sprintf("invalid routing path %q: %s", r.Path, err))
		}
		for _, method := range strings.Split(r.Methods, ",") {
			handlers[method] = append(handlers[method], handler{
				rx:    rx,
				names: names,
				fn:    r.Func,
			})
		}
	}
	return &Router{
		handlers: handlers,
	}
}

type Router struct {
	handlers map[string][]handler // method => route
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rt.ServeCtxHTTP(ctx, w, r)
}

func (rt *Router) ServeCtxHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	for _, h := range rt.handlers[r.Method] {
		match := h.rx.FindAllStringSubmatch(r.URL.Path, 1)
		if len(match) == 0 {
			continue
		}
		values := match[0]

		ctx = context.WithValue(ctx, "router:args", &args{
			names:  h.names,
			values: values[1:],
		})
		h.fn(ctx, w, r)
		return
	}
}

type args struct {
	names  []string
	values []string
}

func (a *args) Len() int {
	return len(a.values)
}

func (a *args) ByName(name string) string {
	for i, n := range a.names {
		if n == name {
			return a.values[i]
		}
	}
	return ""
}

func (a *args) ByIndex(n int) string {
	if len(a.values) < n {
		return a.values[n]
	}
	return ""
}

type handler struct {
	rx    *regexp.Regexp
	names []string
	fn    HandlerFunc
}

func Args(ctx context.Context) PathArgs {
	return ctx.Value("router:args").(*args)
}

type PathArgs interface {
	ByName(string) string
	ByIndex(int) string
}
