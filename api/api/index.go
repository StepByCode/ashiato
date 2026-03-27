package handler

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/dokkiitech/ashiato/api/internal/app"
	"github.com/dokkiitech/ashiato/api/internal/corsutil"
)

var (
	initOnce       sync.Once
	httpHandler    http.Handler
	initErr        error
	allowedOrigins []string
)

func Handler(w http.ResponseWriter, r *http.Request) {
	initOnce.Do(func() {
		httpHandler, _, _, initErr = app.NewHTTPHandler(context.Background())
		allowedOrigins = corsutil.BuildAllowedOrigins(parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS")))
	})

	if initErr != nil {
		http.Error(w, initErr.Error(), http.StatusInternalServerError)
		return
	}

	origin := r.Header.Get("Origin")
	if origin != "" && corsutil.IsOriginAllowed(origin, allowedOrigins) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
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

func parseAllowedOrigins(raw string) []string {
	var origins []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			origins = append(origins, s)
		}
	}
	return origins
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
