package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/Pelfox/go-shortener/internal/middlewares"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/Pelfox/go-shortener/pkg/schemas"
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
	router.Use(middlewares.CompressMiddleware)

	server := &Server{
		addr:    addr,
		prefix:  urlPrefix[strings.LastIndex(urlPrefix, "/")+1:],
		router:  router,
		storage: make(map[string]string),
		mutex:   &sync.RWMutex{},
	}

	router.Post("/", server.handleCreationRequest)
	router.Post("/api/shorten", server.handleShortenRequest)
	router.Get("/*", server.handleShortRequest)

	return server
}

func (s *Server) createShortLink(destinationURL string) (string, error) {
	destinationURL = strings.TrimSpace(destinationURL)
	if destinationURL == "" {
		return "", errors.New("the destination URL is empty")
	}

	shortID := pkg.GenerateShortID(8)
	linkSlug := shortID
	if s.prefix != "" {
		linkSlug = fmt.Sprintf("%s/%s", s.prefix, shortID)
	}

	s.mutex.Lock()
	s.storage[linkSlug] = destinationURL
	s.mutex.Unlock()

	return fmt.Sprintf("http://%s/%s", s.addr, linkSlug), nil
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

	shortLink, err := s.createShortLink(string(body))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create short link: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortLink))

	log.Info().Str("slug", shortLink).
		Str("destination", string(body)).
		Msg("created short URL")
}

func (s *Server) handleShortenRequest(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
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

	var request schemas.CreateShortLink
	if err := json.Unmarshal(body, &request); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal JSON")
		http.Error(w, "Invalid JSON body.", http.StatusBadRequest)
		return
	}

	shortLink, err := s.createShortLink(request.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create short link: %v", err), http.StatusInternalServerError)
		return
	}

	response := schemas.ShortLinkResponse{Result: shortLink}
	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal JSON")
		http.Error(w, "Failed to create response.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(responseBody)

	log.Info().Str("slug", shortLink).
		Str("destination", request.URL).
		Msg("created short URL (via API)")
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
