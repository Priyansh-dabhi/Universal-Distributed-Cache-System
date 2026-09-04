package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"universal-distributed-cache/internal/cache"
)

func BenchmarkHTTPServerGet(b *testing.B) {
	c, _ := cache.NewLRU(1000)
	c.Set("bench-key", "bench-val")
	srv := New(c, DefaultConfig(8080))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := ts.Client()
	url := ts.URL + "/cache/bench-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}
}

func BenchmarkHTTPServerPut(b *testing.B) {
	c, _ := cache.NewLRU(1000)
	srv := New(c, DefaultConfig(8080))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := ts.Client()
	payload := []byte(`{"value":"updated-val"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("put-key-%d", i%500)
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/cache/"+key, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}
}

func BenchmarkHTTPServerDelete(b *testing.B) {
	c, _ := cache.NewLRU(1000)
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("del-key-%d", i), "val")
	}
	srv := New(c, DefaultConfig(8080))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := ts.Client()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("del-key-%d", i%1000)
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/cache/"+key, nil)
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}
}
