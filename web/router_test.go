package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

func TestRouter(t *testing.T) {
	var result struct {
		id     int
		values []string
	}
	testhandler := func(id int, names ...string) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			var values []string
			for _, name := range names {
				values = append(values, Args(ctx).ByName(name))
			}
			result.id = id
			result.values = values
		}
	}

	rt := NewRouter(Routes{
		{"GET", `/x/{w:\w+}/{n:\d+}`, testhandler(11, "w", "n")},
		{"GET", `/x/{n:\d+}/{w:\w+}`, testhandler(12, "w", "n")},
		{"GET", `/x/{n:\d+}-{w:\w+}`, testhandler(13, "w", "n")},

		{"GET", `/x/321`, testhandler(22)},
		{"GET", `/x/{first}`, testhandler(21, "first")},

		{"GET", `/`, testhandler(31)},
		{"GET", `/{a}/{b}`, testhandler(32, "a", "b")},
		{"GET", `/{a}/{b}/{c}`, testhandler(33, "a", "b", "c")},
		{"GET", `/{a}/{b}/{c}/{d}`, testhandler(34, "a", "b", "c", "d")},
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
		if result.values != nil && tc.wantValues != nil {
			if d := diff(result.values, tc.wantValues); len(d) != 0 {
				t.Errorf("%d: want values %#v, got %#v: %v", i, tc.wantValues, result.values, d)
			}
		}

	}
}

func diff(a, b []string) []string {
	var diff []string

	for _, s := range a {
		if !has(b, s) {
			diff = append(diff, s)
		}
	}
	for _, s := range b {
		if !has(a, s) {
			diff = append(diff, s)
		}
	}
	return diff
}

func has(a []string, s string) bool {
	for _, el := range a {
		if el == s {
			return true
		}
	}
	return false
}
