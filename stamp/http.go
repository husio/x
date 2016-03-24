package stamp

import (
	"net/http"

	"golang.org/x/net/context"
)

func X() func(context.Context, http.ResponseWriter, *http.Request) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	}
}
