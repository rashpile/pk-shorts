package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	bolt "go.etcd.io/bbolt"
)

const (
	defaultPrefix     = "/s"
	defaultUIPrefix   = "/sui"
	dbFile            = "links.db"
	bucketName        = "links"
	shortIDLength     = 8
	secureIDLength    = 16
)

type Link struct {
	Short     string    `json:"short"`
	Original  string    `json:"original"`
	CreatedAt time.Time `json:"created_at"`
	Clicks    int       `json:"clicks"`
}

type Server struct {
	db       *bolt.DB
	router   *mux.Router
	prefix   string
	uiPrefix string
	tmpl     *template.Template
}

func NewServer() (*Server, error) {
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	prefix := os.Getenv("SHORT_PREFIX")
	if prefix == "" {
		prefix = defaultPrefix
	}

	uiPrefix := os.Getenv("UI_PREFIX")
	if uiPrefix == "" {
		uiPrefix = defaultUIPrefix
	}

	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Server{
		db:       db,
		prefix:   prefix,
		uiPrefix: uiPrefix,
		tmpl:     tmpl,
	}, nil
}

func (s *Server) Close() error {
	return s.db.Close()
}

func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()

	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	s.router.HandleFunc(s.uiPrefix, s.handleHome).Methods("GET")
	s.router.HandleFunc(s.uiPrefix+"/", s.handleHome).Methods("GET")
	s.router.HandleFunc(s.uiPrefix+"/create", s.handleCreate).Methods("POST")
	s.router.HandleFunc(s.uiPrefix+"/list", s.handleList).Methods("GET")
	s.router.HandleFunc(s.uiPrefix+"/api/create", s.handleAPICreate).Methods("POST")
	s.router.HandleFunc(s.uiPrefix+"/api/list", s.handleAPIList).Methods("GET")
	s.router.HandleFunc(s.uiPrefix+"/api/delete/{short}", s.handleAPIDelete).Methods("DELETE")
	s.router.HandleFunc(s.uiPrefix+"/delete/{short}", s.handleDelete).Methods("POST")

	s.router.HandleFunc(s.prefix+"/{short}", s.handleRedirect).Methods("GET")

	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"UIPrefix": s.uiPrefix,
		"Prefix":   s.prefix,
		"Host":     r.Host,
	}

	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	url := r.FormValue("url")
	if url == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	secure := r.FormValue("secure") == "on"
	customID := strings.TrimSpace(r.FormValue("custom_id"))

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	short, err := s.createShortLink(url, secure, customID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create short link: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"UIPrefix": s.uiPrefix,
		"Prefix":   s.prefix,
		"Host":     r.Host,
		"Success":  true,
		"ShortURL": fmt.Sprintf("http://%s%s/%s", r.Host, s.prefix, short),
		"Original": url,
	}

	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	links, err := s.getAllLinks()
	if err != nil {
		http.Error(w, "Failed to get links", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"UIPrefix": s.uiPrefix,
		"Prefix":   s.prefix,
		"Host":     r.Host,
		"Links":    links,
	}

	if err := s.tmpl.ExecuteTemplate(w, "list.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		log.Printf("Template error: %v", err)
	}
}

func (s *Server) handleAPICreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL      string `json:"url"`
		Secure   bool   `json:"secure"`
		CustomID string `json:"custom_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		req.URL = "https://" + req.URL
	}

	short, err := s.createShortLink(req.URL, req.Secure, strings.TrimSpace(req.CustomID))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create short link: %v", err), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"short":     short,
		"short_url": fmt.Sprintf("http://%s%s/%s", r.Host, s.prefix, short),
		"original":  req.URL,
		"secure":    req.Secure,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleAPIList(w http.ResponseWriter, r *http.Request) {
	links, err := s.getAllLinks()
	if err != nil {
		http.Error(w, "Failed to get links", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(links)
}

func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	short := vars["short"]

	url, err := s.getOriginalURL(short)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	s.incrementClicks(short)

	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	short := vars["short"]

	if err := s.deleteLink(short); err != nil {
		http.Error(w, "Failed to delete link", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, s.uiPrefix+"/list", http.StatusSeeOther)
}

func (s *Server) handleAPIDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	short := vars["short"]

	if err := s.deleteLink(short); err != nil {
		if err.Error() == "link not found" {
			http.Error(w, "Link not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete link", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "short": short})
}

func (s *Server) createShortLink(originalURL string, secure bool, customID string) (string, error) {
	var short string

	// Use custom ID if provided
	if customID != "" {
		// Validate custom ID
		if err := validateCustomID(customID); err != nil {
			return "", err
		}
		short = customID
	} else if secure {
		short = generateSecureID()
	} else {
		short = generateShortID()
	}

	link := Link{
		Short:     short,
		Original:  originalURL,
		CreatedAt: time.Now(),
		Clicks:    0,
	}

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		// Check if custom ID already exists
		if customID != "" {
			existing := b.Get([]byte(short))
			if existing != nil {
				return fmt.Errorf("custom ID '%s' already exists", short)
			}
		} else {
			// For random IDs, keep generating until we find a unique one
			for {
				existing := b.Get([]byte(short))
				if existing == nil {
					break
				}
				if secure {
					short = generateSecureID()
				} else {
					short = generateShortID()
				}
				link.Short = short
			}
		}

		data, err := json.Marshal(link)
		if err != nil {
			return err
		}

		return b.Put([]byte(short), data)
	})

	if err != nil {
		return "", err
	}

	return short, nil
}

func (s *Server) getOriginalURL(short string) (string, error) {
	var link Link

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		data := b.Get([]byte(short))
		if data == nil {
			return fmt.Errorf("link not found")
		}
		return json.Unmarshal(data, &link)
	})

	if err != nil {
		return "", err
	}

	return link.Original, nil
}

func (s *Server) incrementClicks(short string) {
	s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		data := b.Get([]byte(short))
		if data == nil {
			return nil
		}

		var link Link
		if err := json.Unmarshal(data, &link); err != nil {
			return err
		}

		link.Clicks++

		data, err := json.Marshal(link)
		if err != nil {
			return err
		}

		return b.Put([]byte(short), data)
	})
}

func (s *Server) getAllLinks() ([]Link, error) {
	var links []Link

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		return b.ForEach(func(k, v []byte) error {
			var link Link
			if err := json.Unmarshal(v, &link); err != nil {
				return err
			}
			links = append(links, link)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return links, nil
}

func (s *Server) deleteLink(short string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		existing := b.Get([]byte(short))
		if existing == nil {
			return fmt.Errorf("link not found")
		}

		return b.Delete([]byte(short))
	})
}

func validateCustomID(id string) error {
	// Check length
	if len(id) < 3 {
		return fmt.Errorf("custom ID must be at least 3 characters long")
	}
	if len(id) > 50 {
		return fmt.Errorf("custom ID must be no more than 50 characters long")
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	for _, ch := range id {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return fmt.Errorf("custom ID can only contain letters, numbers, dashes, and underscores")
		}
	}

	// Check for reserved words (add more as needed)
	reserved := []string{"api", "admin", "health", "static", "assets", "js", "css"}
	lowerID := strings.ToLower(id)
	for _, r := range reserved {
		if lowerID == r {
			return fmt.Errorf("'%s' is a reserved word and cannot be used as a custom ID", id)
		}
	}

	return nil
}

func generateShortID() string {
	b := make([]byte, shortIDLength)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:shortIDLength]
}

func generateSecureID() string {
	b := make([]byte, secureIDLength)
	rand.Read(b)
	// Use a longer string and replace problematic characters for URL safety
	id := base64.URLEncoding.EncodeToString(b)
	// Remove padding and ensure consistent length
	id = strings.ReplaceAll(id, "=", "")
	id = strings.ReplaceAll(id, "-", "x")
	id = strings.ReplaceAll(id, "_", "y")
	if len(id) > secureIDLength {
		return id[:secureIDLength]
	}
	return id
}

func main() {
	srv, err := NewServer()
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer srv.Close()

	srv.setupRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", port)
		log.Printf("Short link prefix: %s", srv.prefix)
		log.Printf("UI prefix: %s", srv.uiPrefix)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}