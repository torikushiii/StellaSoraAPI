package httpserver

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ss-api/internal/app"
	"ss-api/internal/http/handlers"
)

const (
	colorReset   = "\033[0m"
	colorGreen   = "\033[32m"
	colorCyan    = "\033[36m"
	colorYellow  = "\033[33m"
	colorRed     = "\033[31m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
)

type Server struct {
	mux      *http.ServeMux
	handlers handlers.Set
	logger   *log.Logger
}

func New(appInstance *app.App) *Server {
	mux := http.NewServeMux()
	handlerSet := handlers.New(appInstance)

	srv := &Server{
		mux:      mux,
		handlers: handlerSet,
		logger:   log.New(os.Stdout, "", log.LstdFlags),
	}

	srv.registerRoutes()
	return srv
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}
		s.mux.ServeHTTP(rec, r)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		duration := time.Since(start)
		elapsedMillis := float64(duration) / float64(time.Millisecond)
		if status == http.StatusNotFound && !strings.HasPrefix(r.URL.Path, "/stella") {
			return
		}

		s.logger.Printf(
			"%s%s%s %s%d%s %s %.2fms",
			methodColor(r.Method), r.Method, colorReset,
			statusColor(status), status, colorReset,
			r.URL.Path,
			elapsedMillis,
		)
	})
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /stella", s.handlers.Status)
	s.mux.HandleFunc("GET /stella/", s.handlers.Status)
	s.mux.HandleFunc("GET /stella/characters", s.handlers.Characters)
	s.mux.HandleFunc("GET /stella/character/{identifier}", s.handlers.CharacterDetail)
	s.mux.HandleFunc("GET /stella/discs", s.handlers.Discs)
	s.mux.HandleFunc("GET /stella/disc/{identifier}", s.handlers.DiscDetail)
	s.mux.HandleFunc("GET /stella/banners", s.handlers.Banner)
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func statusColor(status int) string {
	switch {
	case status >= 500:
		return colorRed
	case status >= 400:
		return colorYellow
	case status >= 300:
		return colorCyan
	default:
		return colorGreen
	}
}

func methodColor(method string) string {
	switch method {
	case http.MethodGet:
		return colorBlue
	case http.MethodPost:
		return colorMagenta
	case http.MethodPut:
		return colorYellow
	case http.MethodDelete:
		return colorRed
	case http.MethodPatch:
		return colorCyan
	default:
		return colorGreen
	}
}
