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
}

type cacheInfoResponse struct {
	Size     int    `json:"size"`
	Capacity int    `json:"capacity"`
	Policy   string `json:"policy"`
}

type errorResponse struct {
	Error string `json:"error"`
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
		// Health check endpoint
		if r.URL.Path == "/health" {
			s.handleHealth(w, r)
			return
		}

		// Cache info endpoint
		if r.URL.Path == "/cache" {
			s.handleCacheInfo(w, r)
			return
		}

		// Cache entry endpoint
		if strings.HasPrefix(r.URL.Path, "/cache/") {
			s.handleCacheEntry(w, r)
			return
		}

		// Unmatched route
		writeError(w, http.StatusNotFound, "not found")
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
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
		writeError(w, http.StatusNotFound, "key not found")
		return
	}

	writeJSON(w, http.StatusOK, deleteResponse{
		Message: "cache entry deleted",
	})
}
