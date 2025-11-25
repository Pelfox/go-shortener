package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Pelfox/go-shortener/pkg"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	addr   string
	prefix string

	router  *chi.Mux
	storage map[string]string
}

func NewServer(addr string, urlPrefix string) *Server {
	router := chi.NewRouter()
	server := &Server{
		addr:    addr,
		prefix:  urlPrefix[strings.LastIndex(urlPrefix, "/")+1:],
		router:  router,
		storage: make(map[string]string),
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
	linkSlug := fmt.Sprintf("%s/%s", s.prefix, shortID)
	s.storage[linkSlug] = destinationURL

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("http://%s/%s", s.addr, linkSlug)))
}

func (s *Server) handleShortRequest(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/")
	destinationURL, exists := s.storage[slug]
	if !exists {
		http.Error(w, "Short URL not found.", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, destinationURL, http.StatusTemporaryRedirect)
}

func (s *Server) ServeHTTP() error {
	fmt.Println("Starting server on", s.addr)
	return http.ListenAndServe(s.addr, s.router)
}
