package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/Pelfox/go-shortener/internal/middlewares"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Server struct {
	addr   string
	prefix string

	router  *chi.Mux
	storage map[string]string
	mutex   *sync.RWMutex
}

func NewServer(addr string, urlPrefix string) *Server {
	router := chi.NewRouter()
	router.Use(middlewares.LoggerMiddleware)

	server := &Server{
		addr:    addr,
		prefix:  urlPrefix[strings.LastIndex(urlPrefix, "/")+1:],
		router:  router,
		storage: make(map[string]string),
		mutex:   &sync.RWMutex{},
	}

	router.Post("/", server.handleCreationRequest)
	router.Get("/*", server.handleShortRequest)

	return server
}

func (s *Server) handleCreationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(w, "Invalid content type.", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to read body")
		http.Error(w, "Failed to read request body.", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	destinationURL := strings.TrimSpace(string(body))
	if destinationURL == "" {
		http.Error(w, "The supplied destination URL is empty.", http.StatusBadRequest)
		return
	}

	shortID := pkg.GenerateShortID(8)
	linkSlug := shortID
	if s.prefix != "" {
		linkSlug = fmt.Sprintf("%s/%s", s.prefix, shortID)
	}

	s.mutex.Lock()
	s.storage[linkSlug] = destinationURL
	s.mutex.Unlock()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("http://%s/%s", s.addr, linkSlug)))

	log.Info().Str("slug", linkSlug).
		Str("destination", destinationURL).
		Msg("created short URL")
}

func (s *Server) handleShortRequest(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/")
	s.mutex.RLock()
	destinationURL, exists := s.storage[slug]
	s.mutex.RUnlock()
	if !exists {
		log.Warn().Str("slug", slug).Msg("short URL not found")
		http.Error(w, "Short URL not found.", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, destinationURL, http.StatusTemporaryRedirect)
}

func (s *Server) ServeHTTP() error {
	log.Info().Str("addr", s.addr).Msg("starting server")
	return http.ListenAndServe(s.addr, s.router)
}
