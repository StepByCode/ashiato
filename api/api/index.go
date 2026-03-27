package handler

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/dokkiitech/ashiato/api/internal/app"
)

var (
	initOnce    sync.Once
	httpHandler http.Handler
	initErr     error
)

func Handler(w http.ResponseWriter, r *http.Request) {
	initOnce.Do(func() {
		httpHandler, _, _, initErr = app.NewHTTPHandler(context.Background())
	})

	if initErr != nil {
		http.Error(w, initErr.Error(), http.StatusInternalServerError)
		return
	}

	if originalPath := r.URL.Query().Get("__pathname"); originalPath != "" {
		cloned := r.Clone(r.Context())
		cloned.URL.Path = normalizePath(originalPath)
		query := cloned.URL.Query()
		query.Del("__pathname")
		cloned.URL.RawQuery = query.Encode()
		r = cloned
	}

	httpHandler.ServeHTTP(w, r)
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}
