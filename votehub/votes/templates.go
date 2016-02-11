package votes

import "net/http"

func stdHTMLResp(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
