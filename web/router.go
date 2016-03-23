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

// Route binds together HTTP method, path and handler function.
type Route struct {
	// Method is string that can represent one or more, coma separated HTTP
	// methods that this route should match.
	Methods string
	// Path defines regexp-like pattern match used to determine if route should
	// handle request.
	Path string
	// Func defines HTTP handler that is used to serve request when route is
	// matching.
	Func HandlerFunc
}

// AnyMethod is shortcut definition for
var AnyMethod = "GET,POST,PUT,DELETE"

// NewRouter create and return immutable router instance.
func NewRouter(routes Routes) *Router {
	handlers := make(map[string][]handler)
	builder := regexp.MustCompile("{.*?}")
	// ReplaceAllString

	for _, r := range routes {
		var names []string
		raw := builder.ReplaceAllStringFunc(r.Path, func(s string) string {
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

// ServeHTTP handle HTTP request using empty context.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rt.ServeCtxHTTP(ctx, w, r)
}

// ServeCtxHTTP handle HTTP request using given context.
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

// WithArgs return context with HTTP args set to given list of pairs.
//
// Use for testing only.
func WithArgs(ctx context.Context, pairs ...string) context.Context {
	if len(pairs)%2 != 0 {
		panic("Invalid args list: pairs are not even")
	}
	a := args{}
	for i := 0; i < len(pairs); i += 2 {
		a.names = append(a.names, pairs[i])
		a.values = append(a.values, pairs[i+1])
	}
	return context.WithValue(ctx, "router:args", &a)
}

// Len return number of arguments.
func (a *args) Len() int {
	return len(a.values)
}

// ByName return URL mached value using name assigned to it. Returns empty
// string if does not exist.
func (a *args) ByName(name string) string {
	for i, n := range a.names {
		if n == name {
			return a.values[i]
		}
	}
	return ""
}

// ByIndex return URL mached value using definition position. Returns empty
// string if does not exist.
func (a *args) ByIndex(n int) string {
	if len(a.values) < n {
		return ""
	}
	return a.values[n]
}

type handler struct {
	rx    *regexp.Regexp
	names []string
	fn    HandlerFunc
}

// Args return PathArgs carried by given context.
func Args(ctx context.Context) PathArgs {
	return ctx.Value("router:args").(*args)
}

type PathArgs interface {
	ByName(string) string
	ByIndex(int) string
}
