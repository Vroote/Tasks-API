package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	middleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/example/tasks-api/internal/oapi"
	"github.com/example/tasks-api/internal/store"
)

func New(s *store.MemoryStore) (http.Handler, error) {
	spec, err := oapi.GetSwagger()
	if err != nil {
		return nil, err
	}

	// чтобы в ошибке не было всей схемы целиком
	openapi3.SchemaErrorDetailsDisabled = true

	validator := middleware.OapiRequestValidatorWithOptions(spec, &middleware.Options{
		SilenceServersWarning: true,
		Options: openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, opts middleware.ErrorHandlerOpts) {
			writeError(w, opts.StatusCode, err.Error())
		},
	})

	handler := oapi.NewStrictHandlerWithOptions(&Handlers{store: s}, nil, oapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeError(w, http.StatusBadRequest, err.Error())
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Println("error:", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		},
	})

	mux := http.NewServeMux()
	oapi.HandlerWithOptions(handler, oapi.StdHTTPServerOptions{
		BaseURL:     "/api/v1",
		BaseRouter:  mux,
		Middlewares: []oapi.MiddlewareFunc{validator},
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeError(w, http.StatusBadRequest, err.Error())
		},
	})

	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spec)
	})
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(swaggerPage))
	})

	return logRequests(mux), nil
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(oapi.Error{Message: msg})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

const swaggerPage = `<!DOCTYPE html>
<html>
<head>
  <title>Tasks API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>SwaggerUIBundle({ url: "/openapi.json", dom_id: "#swagger-ui" })</script>
</body>
</html>`
