package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Pelfox/go-shortener/pkg"
)

type Server struct {
	addr    string
	mux     *http.ServeMux
	storage map[string]string
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	server := &Server{
		addr:    addr,
		mux:     mux,
		storage: make(map[string]string),
	}

	mux.HandleFunc("POST /", server.handleCreationRequest)
	mux.HandleFunc("GET /{id}", server.handleShortRequest)

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
	s.storage[shortID] = destinationURL

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("http://%s/%s", s.addr, shortID)))
}

func (s *Server) handleShortRequest(w http.ResponseWriter, r *http.Request) {
	params := r.PathValue("id")

	destinationURL, exists := s.storage[params]
	if !exists {
		http.Error(w, "Short URL not found.", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, destinationURL, http.StatusTemporaryRedirect)
}

func (s *Server) ServeHTTP() error {
	return http.ListenAndServe(s.addr, s.mux)
}
