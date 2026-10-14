package supabase

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerCappedPagination(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("apikey") != "test-key" || r.Header.Get("Authorization") != "Bearer test-key" || r.Method != "GET" {
			t.Error("unexpected auth/method")
		}
		if r.URL.Path != "/rest/v1/playerauctions_services" || r.URL.Query().Get("order") != "id.asc" {
			t.Error("unexpected query")
		}
		switch r.URL.Query().Get("offset") {
		case "0":
			fmt.Fprint(w, `[{"id":1}]`)
		case "1":
			w.WriteHeader(206)
			fmt.Fprint(w, `[{"id":2}]`)
		case "2":
			fmt.Fprint(w, `[]`)
		default:
			t.Error("unexpected offset")
			fmt.Fprint(w, `[]`)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	services, err := client.Services(context.Background())
	if err != nil || len(services) != 2 || calls != 3 {
		t.Fatalf("services=%v calls=%d err=%v", services, calls, err)
	}
}

func TestSharedDescription(t *testing.T) {
	for _, test := range []struct {
		body      string
		wantError bool
	}{
		{`[{"content":"Shared HTML"}]`, false},
		{`[]`, true},
		{`[{"content":"one"},{"content":"two"}]`, true},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/rest/v1/playerauctions_descriptions" || r.URL.Query().Get("select") != "content" {
				t.Error("unexpected query")
			}
			if r.URL.Query().Get("offset") != "0" {
				fmt.Fprint(w, `[]`)
				return
			}
			fmt.Fprint(w, test.body)
		}))
		client, err := NewClient(server.URL, "test-key")
		if err != nil {
			t.Fatal(err)
		}
		content, err := client.SharedDescription(context.Background())
		server.Close()
		if (err != nil) != test.wantError {
			t.Fatalf("content=%q err=%v", content, err)
		}
		if !test.wantError && content != "Shared HTML" {
			t.Fatal(content)
		}
	}
}

func TestFetchFailures(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
	}{{401, `denied`}, {200, `invalid json`}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(test.status); fmt.Fprint(w, test.body) }))
		client, err := NewClient(server.URL, "test-key")
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.SharedDescription(context.Background())
		server.Close()
		if err == nil {
			t.Fatal("expected error")
		}
	}
}
