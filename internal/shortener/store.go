package shortener

import (
	"sync"
)

// Store maneja el almacenamiento concurrente de URLs
type Store struct {
	urls map[string]string // short_code -> long_url
	mu   sync.RWMutex      // Mutex para operaciones concurrentes
}

// NewStore crea una nueva instancia del almacén
func NewStore() *Store {
	return &Store{
		urls: make(map[string]string),
	}
}

// Save almacena una nueva relación short_code -> long_url
func (s *Store) Save(shortCode, longURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[shortCode] = longURL
}

// Get obtiene la URL larga asociada a un código corto
func (s *Store) Get(shortCode string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	longURL, exists := s.urls[shortCode]
	return longURL, exists
}

// Exists verifica si un código corto ya existe
func (s *Store) Exists(shortCode string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.urls[shortCode]
	return exists
}

// Count retorna el número total de URLs almacenadas
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.urls)
}
