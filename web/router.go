package web

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/context"
)

type Routes []Route

type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

type Route struct {
	Path    string
	Name    string
	Methods string
	Func    HandlerFunc
}

func GET(path, name string, fn HandlerFunc) Route {
	return Route{
		Path:    path,
		Name:    name,
		Methods: "GET",
		Func:    fn,
	}
}

func POST(path, name string, fn HandlerFunc) Route {
	return Route{
		Path:    path,
		Name:    name,
		Methods: "POST",
		Func:    fn,
	}
}

func PUT(path, name string, fn HandlerFunc) Route {
	return Route{
		Path:    path,
		Name:    name,
		Methods: "PUT",
		Func:    fn,
	}
}

func DELETE(path, name string, fn HandlerFunc) Route {
	return Route{
		Path:    path,
		Name:    name,
		Methods: "DELETE",
		Func:    fn,
	}
}

func ANY(path, name string, fn HandlerFunc) Route {
	return Route{
		Path:    path,
		Name:    name,
		Methods: "GET,POST,PUT,DELETE",
		Func:    fn,
	}
}

func NewRouter(prefix string, routes Routes) *Router {
	paths := make(map[string]pathtemplate)
	handlers := make(map[string][]handler)

	parser := regexp.MustCompile("{.*?}")

	for _, r := range routes {
		var names []string
		raw := parser.ReplaceAllStringFunc(prefix+r.Path, func(s string) string {
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
		methods := strings.Split(strings.TrimSpace(r.Methods), ",")
		for _, method := range methods {
			handlers[method] = append(handlers[method], handler{
				handlerName: r.Name,
				rx:          rx,
				names:       names,
				fn:          r.Func,
			})
		}

		if r.Name != "" {
			t := parser.ReplaceAllString(r.Path, "%v")
			if tp, ok := paths[r.Name]; ok && t != tp.tmpl {
				log.Printf("router name duplicate: %s (%v => %v)", r.Name, t, tp.tmpl)
			}
			paths[r.Name] = pathtemplate{
				tmpl: t,
				argc: len(names),
			}
		}
	}
	return &Router{
		handlers: handlers,
		paths:    paths,
	}
}

type Router struct {
	handlers map[string][]handler    // method => route
	paths    map[string]pathtemplate // name => url template
}

type pathtemplate struct {
	tmpl string
	argc int
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
			handlerName: h.handlerName,
			names:       h.names,
			values:      values[1:],
		})
		h.fn(ctx, w, r)
		return
	}
}

func (rt *Router) Reverse(name string, args ...interface{}) (string, error) {
	t, ok := rt.paths[name]
	if !ok {
		return "", ErrRouteNotFound
	}
	if len(args) != t.argc {
		return "", fmt.Errorf("%d arguments expected", t.argc)
	}
	return fmt.Sprintf(t.tmpl, args...), nil
}

var ErrRouteNotFound = errors.New("route not found")

type args struct {
	handlerName string
	names       []string
	values      []string
}

func (a *args) Len() int {
	return len(a.values)
}

func (a *args) HandlerName() string {
	return a.handlerName
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
		return ""
	}
	return a.values[n]
}

type handler struct {
	rx          *regexp.Regexp
	handlerName string
	names       []string
	fn          HandlerFunc
}

func Args(ctx context.Context) PathArgs {
	return ctx.Value("router:args").(*args)
}

type PathArgs interface {
	HandlerName() string
	ByName(string) string
	ByIndex(int) string
}

func WithRouter(ctx context.Context, rt *Router) context.Context {
	return context.WithValue(ctx, "web:router", rt)
}

func Reverse(ctx context.Context, name string, args ...interface{}) string {
	v := ctx.Value("web:router")
	if v == nil {
		log.Print("router not present in context")
		return ""
	}
	path, err := v.(*Router).Reverse(name, args...)
	if err != nil {
		log.Printf("cannot reverse %s: %s", name, err)
		return ""
	}
	return path
}
