package handlers

import (
	"net/http"
)

// RedirectHandler returns an HTTP handler function for short URLs
func RedirectHandler(getLongURL func(shortKey string) (string, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortKey := r.URL.Path[1:] // remove leading "/"

		if longURL, ok := getLongURL(shortKey); ok {
			http.Redirect(w, r, longURL, http.StatusFound) // 302 redirect
			return
		}

		http.NotFound(w, r)
	}
}
