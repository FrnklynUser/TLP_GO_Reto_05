package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"acortador-urls/internal/shortener"
)

func TestHandler_ShortenURL(t *testing.T) {
	store := shortener.NewStore()
	service := shortener.NewService(store)
	handler := NewHandler(service)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectShortURL bool
	}{
		{
			name:           "URL válida",
			requestBody:    `{"long_url": "https://www.example.com/very/long/path?param=value"}`,
			expectedStatus: http.StatusCreated,
			expectShortURL: true,
		},
		{
			name:           "URL con HTTP",
			requestBody:    `{"long_url": "http://example.com"}`,
			expectedStatus: http.StatusCreated,
			expectShortURL: true,
		},
		{
			name:           "JSON inválido",
			requestBody:    `{"long_url": "https://example.com"`,
			expectedStatus: http.StatusBadRequest,
			expectShortURL: false,
		},
		{
			name:           "URL vacía",
			requestBody:    `{"long_url": ""}`,
			expectedStatus: http.StatusBadRequest,
			expectShortURL: false,
		},
		{
			name:           "URL solo espacios",
			requestBody:    `{"long_url": "   "}`,
			expectedStatus: http.StatusBadRequest,
			expectShortURL: false,
		},
		{
			name:           "URL sin esquema",
			requestBody:    `{"long_url": "www.example.com"}`,
			expectedStatus: http.StatusBadRequest,
			expectShortURL: false,
		},
		{
			name:           "URL inválida",
			requestBody:    `{"long_url": "not-a-valid-url"}`,
			expectedStatus: http.StatusBadRequest,
			expectShortURL: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.ShortenURL(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectShortURL {
				var response ShortenResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Errorf("Error decoding response: %v", err)
				}

				if response.ShortURL == "" {
					t.Error("Expected short_url in response")
				}

				if !strings.Contains(response.ShortURL, "http") {
					t.Error("Short URL should be a complete URL")
				}
			} else {
				var errorResponse ErrorResponse
				if err := json.NewDecoder(rr.Body).Decode(&errorResponse); err != nil {
					t.Errorf("Error decoding error response: %v", err)
				}

				if errorResponse.Error == "" {
					t.Error("Expected error field in error response")
				}
			}
		})
	}
}

func TestHandler_RedirectURL(t *testing.T) {
	store := shortener.NewStore()
	service := shortener.NewService(store)
	handler := NewHandler(service)

	// Crear una URL de prueba
	testURL := "https://www.example.com/test"
	shortCode, err := service.ShortenURL(testURL)
	if err != nil {
		t.Fatalf("Error creating test URL: %v", err)
	}

	tests := []struct {
		name           string
		shortCode      string
		expectedStatus int
		expectedURL    string
	}{
		{
			name:           "Código válido",
			shortCode:      shortCode,
			expectedStatus: http.StatusTemporaryRedirect,
			expectedURL:    testURL,
		},
		{
			name:           "Código no existente",
			shortCode:      "nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedURL:    "",
		},
		{
			name:           "Código vacío",
			shortCode:      "",
			expectedStatus: http.StatusBadRequest,
			expectedURL:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configurar router con chi para manejar parámetros de URL
			r := chi.NewRouter()
			r.Get("/{short_code}", handler.RedirectURL)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.shortCode, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusTemporaryRedirect {
				location := rr.Header().Get("Location")
				if location != tt.expectedURL {
					t.Errorf("Expected Location header %s, got %s", tt.expectedURL, location)
				}
			}
		})
	}
}

func TestHandler_Integration(t *testing.T) {
	store := shortener.NewStore()
	service := shortener.NewService(store)
	handler := NewHandler(service)

	// Configurar router completo
	r := chi.NewRouter()
	r.Post("/shorten", handler.ShortenURL)
	r.Get("/{short_code}", handler.RedirectURL)

	// Test flujo completo: acortar y luego redirigir
	testURL := "https://www.example.com/integration/test?param=value"

	// 1. Acortar URL
	requestBody := `{"long_url": "` + testURL + `"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status %d for shorten, got %d", http.StatusCreated, rr.Code)
	}

	var response ShortenResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Error decoding shorten response: %v", err)
	}

	// Extraer el código corto de la URL
	parts := strings.Split(response.ShortURL, "/")
	shortCode := parts[len(parts)-1]

	// 2. Redirigir usando el código corto
	req = httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected status %d for redirect, got %d", http.StatusTemporaryRedirect, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != testURL {
		t.Errorf("Expected redirect to %s, got %s", testURL, location)
	}
}

func TestHandler_ConcurrentRequests(t *testing.T) {
	store := shortener.NewStore()
	service := shortener.NewService(store)
	handler := NewHandler(service)

	r := chi.NewRouter()
	r.Post("/shorten", handler.ShortenURL)
	r.Get("/{short_code}", handler.RedirectURL)

	// Crear servidor de prueba
	server := httptest.NewServer(r)
	defer server.Close()

	// Realizar múltiples peticiones concurrentes
	numRequests := 50
	results := make(chan string, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			testURL := fmt.Sprintf("https://concurrent%d.com", id)
			requestBody := fmt.Sprintf(`{"long_url": "%s"}`, testURL)

			resp, err := http.Post(server.URL+"/shorten", "application/json", strings.NewReader(requestBody))
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				errors <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				return
			}

			var response ShortenResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				errors <- err
				return
			}

			results <- response.ShortURL
		}(i)
	}

	// Recopilar resultados
	shortURLs := make(map[string]bool)
	for i := 0; i < numRequests; i++ {
		select {
		case url := <-results:
			if shortURLs[url] {
				t.Errorf("Duplicate short URL generated: %s", url)
			}
			shortURLs[url] = true
		case err := <-errors:
			t.Errorf("Error in concurrent request: %v", err)
		}
	}

	if len(shortURLs) != numRequests {
		t.Errorf("Expected %d unique URLs, got %d", numRequests, len(shortURLs))
	}
}

func BenchmarkHandler_ShortenURL(b *testing.B) {
	store := shortener.NewStore()
	service := shortener.NewService(store)
	handler := NewHandler(service)

	requestBody := `{"long_url": "https://www.example.com/benchmark/test"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ShortenURL(rr, req)
	}
}

func BenchmarkHandler_RedirectURL(b *testing.B) {
	store := shortener.NewStore()
	service := shortener.NewService(store)
	handler := NewHandler(service)

	// Preparar datos de prueba
	testURL := "https://www.example.com/benchmark"
	shortCode, _ := service.ShortenURL(testURL)

	r := chi.NewRouter()
	r.Get("/{short_code}", handler.RedirectURL)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)
	}
}
