package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/example/tasks-api/internal/oapi"
	"github.com/example/tasks-api/internal/store"
)

type testClient struct {
	t      *testing.T
	srv    *httptest.Server
	router routers.Router
}

func newTestClient(t *testing.T) *testClient {
	log.SetOutput(io.Discard)

	h, err := New(store.NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	spec, _ := oapi.GetSwagger()
	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		t.Fatal(err)
	}
	return &testClient{t: t, srv: srv, router: router}
}

// do отправляет запрос и проверяет, что ответ соответствует openapi.yaml
func (c *testClient) do(method, path, body string) (*http.Response, []byte) {
	c.t.Helper()

	req, _ := http.NewRequest(method, c.srv.URL+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	route, params, err := c.router.FindRoute(req)
	if err != nil {
		c.t.Fatalf("route %s %s not found in spec", method, path)
	}
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: params,
			Route:      route,
		},
		Status:  resp.StatusCode,
		Header:  resp.Header,
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	input.SetBodyBytes(data)
	if err := openapi3filter.ValidateResponse(context.Background(), input); err != nil {
		c.t.Errorf("response doesn't match spec: %v", err)
	}
	return resp, data
}

func TestSpecValid(t *testing.T) {
	spec, err := oapi.GetSwagger()
	if err != nil {
		t.Fatal(err)
	}
	if err := spec.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAndGet(t *testing.T) {
	c := newTestClient(t)

	resp, body := c.do("POST", "/api/v1/tasks", `{"title":"Сдать лабу","dueDate":"2026-11-15"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var created oapi.Task
	json.Unmarshal(body, &created)
	if created.Status != oapi.Todo {
		t.Errorf("default status should be todo, got %s", created.Status)
	}

	location := resp.Header.Get("Location")
	if location != "/api/v1/tasks/"+created.Id.String() {
		t.Errorf("wrong Location header: %s", location)
	}

	resp, body = c.do("GET", location, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var got oapi.Task
	json.Unmarshal(body, &got)
	if got.Id != created.Id || got.Title != "Сдать лабу" {
		t.Errorf("got wrong task: %+v", got)
	}
}

func TestCreateInvalid(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"no title", `{}`},
		{"empty title", `{"title":""}`},
		{"extra field", `{"title":"a","priority":1}`},
		{"bad status", `{"title":"a","status":"archived"}`},
		{"bad date", `{"title":"a","dueDate":"15.11.2026"}`},
		{"broken json", `{"title":`},
	}

	c := newTestClient(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := c.do("POST", "/api/v1/tasks", tt.body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", resp.StatusCode, body)
			}
		})
	}
}

func TestGetErrors(t *testing.T) {
	c := newTestClient(t)

	resp, _ := c.do("GET", "/api/v1/tasks/3f6c2a4e-8b1d-4c7a-9e2f-1a2b3c4d5e6f", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	resp, _ = c.do("GET", "/api/v1/tasks/123", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestList(t *testing.T) {
	c := newTestClient(t)
	c.do("POST", "/api/v1/tasks", `{"title":"a"}`)
	c.do("POST", "/api/v1/tasks", `{"title":"b","status":"done"}`)
	c.do("POST", "/api/v1/tasks", `{"title":"c"}`)

	tests := []struct {
		query      string
		wantTotal  int
		wantTitles []string
	}{
		{"", 3, []string{"a", "b", "c"}},
		{"?status=done", 1, []string{"b"}},
		{"?limit=1&offset=1", 3, []string{"b"}},
		{"?offset=10", 3, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			_, body := c.do("GET", "/api/v1/tasks"+tt.query, "")
			var list oapi.TaskList
			json.Unmarshal(body, &list)

			if list.Total != tt.wantTotal || len(list.Items) != len(tt.wantTitles) {
				t.Fatalf("got total=%d items=%d", list.Total, len(list.Items))
			}
			for i, title := range tt.wantTitles {
				if list.Items[i].Title != title {
					t.Errorf("item %d: expected %s, got %s", i, title, list.Items[i].Title)
				}
			}
		})
	}

	for _, q := range []string{"limit=0", "limit=101", "offset=-1", "status=xxx"} {
		resp, _ := c.do("GET", "/api/v1/tasks?"+q, "")
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", q, resp.StatusCode)
		}
	}
}
