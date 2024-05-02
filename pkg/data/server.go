package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type Server struct {
	sync.RWMutex
	Databases map[string]*Database
}

func NewServer() *Server {
	return &Server{
		Databases: make(map[string]*Database),
	}
}

func (s *Server) Initialize() error {
	serverDir := getDefaultServerDir()
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return fmt.Errorf("failed to create or access server directory: %v", err)
	}

	return s.LoadDatabases(serverDir)
}

func (s *Server) LoadDatabases(serverDir string) error {
	dbs, err := os.ReadDir(serverDir)
	if err != nil {
		return fmt.Errorf("failed to read server directory: %v", err)
	}

	for _, dbInfo := range dbs {
		if dbInfo.IsDir() {
			dbDir := filepath.Join(serverDir, dbInfo.Name())
			db := NewDatabase(dbInfo.Name())
			if err := db.LoadTables(dbDir); err != nil {
				return err
			}
			s.Databases[dbInfo.Name()] = db
		}
	}
	return nil
}

func getDefaultServerDir() string {
	return "./databaseprototype"
}

func (s *Server) CreateDatabase(name string) error {
	s.Lock()
	defer s.Unlock()
	if _, exists := s.Databases[name]; exists {
		return fmt.Errorf("Database %s already exists", name)
	}
	s.Databases[name] = NewDatabase(name)
	return nil
}

func (s *Server) ListDatabases() []string {
	s.RLock()
	defer s.RUnlock()
	var databases []string
	for name := range s.Databases {
		databases = append(databases, name)
	}
	return databases
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		if r.URL.Path == "/createDatabase" {
			var data struct {
				Name string
			}
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := s.CreateDatabase(data.Name); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Fprintf(w, "Database '%s' created successfully", data.Name)
			return
		}
	case "GET":
		if r.URL.Path == "/listDatabases" {
			databases := s.ListDatabases()
			resp, err := json.Marshal(databases)
			if err != nil {
				http.Error(w, "Failed to serialize response", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}
	default:
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}