package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecutor_getContainerForInstance_containerName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances/inst-1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"inst-1","container_name":"c1","status":"running"}`))
	}))
	defer srv.Close()

	e := &Executor{
		config:     Config{APIServerURL: srv.URL},
		httpClient: srv.Client(),
	}

	got, err := e.getContainerForInstance(context.Background(), "inst-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "c1" {
		t.Fatalf("container=%q want %q", got, "c1")
	}
}

func TestExecutor_getContainerForInstance_backwardCompatContainer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances/inst-2" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"inst-2","container":"c2","status":"running"}`))
	}))
	defer srv.Close()

	e := &Executor{
		config:     Config{APIServerURL: srv.URL},
		httpClient: srv.Client(),
	}

	got, err := e.getContainerForInstance(context.Background(), "inst-2")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "c2" {
		t.Fatalf("container=%q want %q", got, "c2")
	}
}

func TestExecutor_getContainerFromAPI_prefersRunning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"instances":[{"id":"i1","container_name":"c1","status":"stopped"},{"id":"i2","container_name":"c2","status":"running"}]}`))
	}))
	defer srv.Close()

	e := &Executor{
		config:     Config{APIServerURL: srv.URL},
		httpClient: srv.Client(),
	}

	got, err := e.getContainerFromAPI(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "c2" {
		t.Fatalf("container=%q want %q", got, "c2")
	}
}

func TestExecutor_getContainerFromAPI_fallbackFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"instances":[{"id":"i1","container_name":"c1","status":"stopped"},{"id":"i2","container_name":"c2","status":"stopped"}]}`))
	}))
	defer srv.Close()

	e := &Executor{
		config:     Config{APIServerURL: srv.URL},
		httpClient: srv.Client(),
	}

	got, err := e.getContainerFromAPI(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "c1" {
		t.Fatalf("container=%q want %q", got, "c1")
	}
}

func TestExecutor_getContainerFromAPI_empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"instances":[]}`))
	}))
	defer srv.Close()

	e := &Executor{
		config:     Config{APIServerURL: srv.URL},
		httpClient: srv.Client(),
	}

	_, err := e.getContainerFromAPI(context.Background(), "acc-1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

