package portalapi

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Options struct {
	PollInterval, HeartbeatInterval, WriteTimeout, ReadTimeout time.Duration
	MaxStreams                                                 int
	Assets                                                     fs.FS
}

type server struct {
	reader  Reader
	options Options
	slots   chan struct{}
}

func NewHandler(reader Reader, options Options) http.Handler {
	if options.PollInterval <= 0 {
		options.PollInterval = time.Second
	}
	if options.HeartbeatInterval <= 0 {
		options.HeartbeatInterval = 15 * time.Second
	}
	if options.WriteTimeout <= 0 {
		options.WriteTimeout = 5 * time.Second
	}
	if options.ReadTimeout <= 0 {
		options.ReadTimeout = 5 * time.Second
	}
	if options.MaxStreams <= 0 {
		options.MaxStreams = 16
	}
	s := &server{reader: reader, options: options, slots: make(chan struct{}, options.MaxStreams)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/projects", s.projects)
	mux.HandleFunc("GET /api/v1/projects/{project}/board", s.board)
	mux.HandleFunc("GET /api/v1/projects/{project}/events", s.events)
	mux.HandleFunc("GET /api/v1/projects/{project}/tickets/{ticket}/attachments/{attachment}", s.attachment)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { apiError(w, 404, "NOT_FOUND", "Resource not found") })
	if options.Assets != nil {
		mux.Handle("/", http.FileServerFS(options.Assets))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { apiError(w, 404, "NOT_FOUND", "Resource not found") })
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		hasBody := r.ContentLength != 0 || len(r.TransferEncoding) != 0
		if hasBody {
			// Closing avoids net/http draining an unfinished body before flushing.
			w.Header().Set("Connection", "close")
			r.Close = true
		}
		if !localHost(r.Host) || !sameOrigin(r) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			apiError(w, 403, "FORBIDDEN", "Only local same-origin access is supported")
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			apiError(w, 405, "METHOD_NOT_ALLOWED", "Only GET is supported")
			return
		}
		if hasBody {
			apiError(w, 400, "REQUEST_BODY_NOT_ALLOWED", "GET requests must not have a body")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// ValidateListen rejects wildcard and network-facing listeners. localhost remains
// valid through SSH port forwarding, which can choose a different local port.
func ValidateListen(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil || !localHost(host) {
		return errors.New("listen must be a loopback host:port")
	}
	return nil
}
func localHost(authority string) bool {
	host := authority
	if h, _, err := net.SplitHostPort(authority); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Scheme == "http" && strings.EqualFold(u.Host, r.Host) && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}
func apiJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func apiError(w http.ResponseWriter, status int, code, message string) {
	apiJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func readError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		apiError(w, 404, "NOT_FOUND", "Project not found")
		return
	}
	apiError(w, 503, "STORE_UNAVAILABLE", "Conductor data is unavailable")
}
func (s *server) projects(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.options.ReadTimeout)
	defer cancel()
	p, err := s.reader.Projects(ctx)
	if err != nil {
		readError(w, err)
		return
	}
	if p == nil {
		p = []Project{}
	}
	apiJSON(w, 200, map[string]any{"projects": p})
}
func (s *server) snapshot(ctx context.Context, id string) (Board, error) {
	ctx, cancel := context.WithTimeout(ctx, s.options.ReadTimeout)
	defer cancel()
	return s.reader.Board(ctx, id)
}
func (s *server) board(w http.ResponseWriter, r *http.Request) {
	b, err := s.snapshot(r.Context(), r.PathValue("project"))
	if err != nil {
		readError(w, err)
		return
	}
	apiJSON(w, 200, b)
}
