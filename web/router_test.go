package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"golang.org/x/net/context"
)

func TestRouter(t *testing.T) {
	var result struct {
		handlerName string
		id          int
		values      []string
	}
	testhandler := func(id int, names ...string) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			var values []string
			args := Args(ctx)
			for _, name := range names {
				values = append(values, args.ByName(name))
			}
			result.id = id
			result.values = values
		}
	}

	rt := NewRouter("", Routes{
		GET(`/x/{w:\w+}/{n:\d+}`, "h11", testhandler(11, "w", "n")),
		GET(`/x/{n:\d+}/{w:\w+}`, "h12", testhandler(12, "n", "w")),
		GET(`/x/{n:\d+}-{w:\w+}`, "h13", testhandler(13, "n", "w")),

		GET(`/x/321`, "h21", testhandler(22)),
		GET(`/x/{first}`, "h22", testhandler(21, "first")),

		GET(`/`, "h31", testhandler(31)),
		GET(`/{a}/{b}`, "h32", testhandler(32, "a", "b")),
		GET(`/{a}/{b}/{c}`, "h33", testhandler(33, "a", "b", "c")),
		GET(`/{a}/{b}/{c}/{d}`, "h34", testhandler(34, "a", "b", "c", "d")),
	})

	var testCases = []struct {
		method     string
		path       string
		wantID     int
		wantValues []string
	}{
		{"GET", "/", 31, nil},
		{"POST", "/", 0, nil},
		{"GET", "/foo/bar", 32, []string{"foo", "bar"}},
		{"GET", "/foo/bar/baz", 33, []string{"foo", "bar", "baz"}},
		{"GET", "/x/x/x/x/x/x/x/x/x/x/x", 0, nil},

		{"GET", "/x/33", 21, []string{"33"}},
		{"GET", "/x/321", 22, nil},

		{"GET", "/x/foo/321", 11, []string{"foo", "321"}},
		{"GET", "/x/321/foo", 12, []string{"321", "foo"}},
		{"GET", "/x/123-321", 13, []string{"123", "321"}},
	}

	for i, tc := range testCases {
		result.id = 0
		result.values = nil

		r, err := http.NewRequest(tc.method, tc.path, nil)
		if err != nil {
			t.Fatalf("%d: cannot create request: %s", i, err)
		}
		rt.ServeHTTP(httptest.NewRecorder(), r)
		if result.id != tc.wantID {
			t.Errorf("%d: want result %d, got %d", i, tc.wantID, result.id)
		}
		if !reflect.DeepEqual(result.values, tc.wantValues) && result.values != nil && tc.wantValues != nil {
			t.Errorf("%d: want values %#v, got %#v", i, tc.wantValues, result.values)
		}
	}
}

func TestRouterReverse(t *testing.T) {
	noph := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {}

	rt := NewRouter("", Routes{
		GET(`/A/{a:\d+}/B/{b}`, "first", noph),
		GET(`/A`, "second", noph),
		GET(`/A/{a:\d+}`, "third", noph),
	})

	var testCases = []struct {
		name string
		args []interface{}
		want string
	}{
		{"first", []interface{}{"1", "2"}, "/A/1/B/2"},
		{"second", nil, "/A"},
		{"third", []interface{}{"foo"}, "/A/foo"}, // reverse does not validate args
	}

	for i, tc := range testCases {
		if got, err := rt.Reverse(tc.name, tc.args...); err != nil {
			t.Errorf("%d (%s): unexpected error: %s", i, tc.name, err)
		} else if tc.want != got {
			t.Errorf("%d (%s): want %q, got %q", i, tc.name, tc.want, got)
		}
	}

}
