package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type setRequest struct {
	Value *string `json:"value"`
	TTL   string  `json:"ttl,omitempty"`
}

type setResponse struct {
	Message string `json:"message"`
}

type getResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type deleteResponse struct {
	Message string `json:"message"`
}

type healthResponse struct {
	Status string `json:"status"`
	NodeID string `json:"node_id,omitempty"`
}

type nodeInfoResponse struct {
	ID       string `json:"id"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Capacity int    `json:"capacity"`
	Policy   string `json:"policy"`
}

type cacheInfoResponse struct {
	Size     int    `json:"size"`
	Capacity int    `json:"capacity"`
	Policy   string `json:"policy"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// statusResponseWriter captures the HTTP status code written by handlers for telemetry.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func (s *Server) routes() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		srw := &statusResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		defer func() {
			s.httpMetrics.RecordRequest(srw.statusCode, time.Since(start))
		}()

		// Health check endpoint
		if r.URL.Path == "/health" {
			s.handleHealth(srw, r)
			return
		}

		// Node info endpoint
		if r.URL.Path == "/node" {
			s.handleNodeInfo(srw, r)
			return
		}

		// Cache info endpoint
		if r.URL.Path == "/cache" {
			s.handleCacheInfo(srw, r)
			return
		}

		// Metrics endpoint
		if r.URL.Path == "/metrics" {
			s.handleMetrics(srw, r)
			return
		}

		// Cache entry endpoint
		if strings.HasPrefix(r.URL.Path, "/cache/") {
			s.handleCacheEntry(srw, r)
			return
		}

		// Unmatched route
		writeError(srw, http.StatusNotFound, "not found")
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{
		Status: "ok",
		NodeID: s.cfg.NodeID,
	})
}

func (s *Server) handleNodeInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, nodeInfoResponse{
		ID:       s.cfg.NodeID,
		Host:     s.cfg.Host,
		Port:     s.cfg.Port,
		Capacity: s.cache.Capacity(),
		Policy:   strings.ToLower(string(s.cache.Policy())),
	})
}

func (s *Server) handleCacheInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, cacheInfoResponse{
		Size:     s.cache.Size(),
		Capacity: s.cache.Capacity(),
		Policy:   strings.ToLower(string(s.cache.Policy())),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	cacheSnap := s.cache.Metrics().Snapshot()
	httpSnap := s.httpMetrics.Snapshot()

	response := map[string]interface{}{
		// Cache engine statistics
		"hits":      cacheSnap.Hits,
		"misses":    cacheSnap.Misses,
		"hit_rate":  cacheSnap.HitRate,
		"sets":      cacheSnap.Sets,
		"deletes":   cacheSnap.Deletes,
		"evictions": cacheSnap.Evictions,
		"expired":   cacheSnap.Expired,

		// Node HTTP statistics
		"requests":       httpSnap.Requests,
		"errors_4xx":     httpSnap.Errors4xx,
		"errors_5xx":     httpSnap.Errors5xx,
		"avg_latency_ms": httpSnap.AvgLatencyMs,

		// Cache state
		"policy":   strings.ToLower(string(s.cache.Policy())),
		"capacity": s.cache.Capacity(),
		"size":     s.cache.Size(),
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCacheEntry(w http.ResponseWriter, r *http.Request) {
	var rawKey string
	if r.URL.RawPath != "" {
		rawKey = strings.TrimPrefix(r.URL.RawPath, "/cache/")
	} else {
		rawKey = strings.TrimPrefix(r.URL.Path, "/cache/")
	}

	key, err := url.PathUnescape(rawKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid URL encoding in key")
		return
	}

	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r, key)
	case http.MethodPut:
		s.handlePut(w, r, key)
	case http.MethodDelete:
		s.handleDelete(w, r, key)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, key string) {
	val, ok := s.cache.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "cache miss")
		return
	}

	writeJSON(w, http.StatusOK, getResponse{
		Key:   key,
		Value: val,
	})
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request, key string) {
	// Limit request body to 1MB to prevent resource exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req setRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body cannot be empty")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Value == nil {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}

	if req.TTL != "" {
		ttl, err := time.ParseDuration(req.TTL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid ttl format")
			return
		}
		s.cache.SetWithTTL(key, *req.Value, ttl)
	} else {
		s.cache.Set(key, *req.Value)
	}

	writeJSON(w, http.StatusOK, setResponse{
		Message: "cache entry stored",
	})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request, key string) {
	deleted := s.cache.Delete(key)
	if !deleted {
		writeError(w, http.StatusNotFound, "cache miss")
		return
	}

	writeJSON(w, http.StatusOK, deleteResponse{
		Message: "cache entry deleted",
	})
}
