package shortener

import (
	"fmt"
	"sync"
	"testing"
)

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewStore()

	// Número de goroutines concurrentes
	numGoroutines := 100
	numOperations := 10

	var wg sync.WaitGroup

	// Test escrituras concurrentes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				shortCode := fmt.Sprintf("code%d_%d", id, j)
				longURL := fmt.Sprintf("https://example.com/%d/%d", id, j)
				store.Save(shortCode, longURL)
			}
		}(i)
	}

	wg.Wait()

	// Verificar que todas las URLs se guardaron
	expectedCount := numGoroutines * numOperations
	if store.Count() != expectedCount {
		t.Errorf("Expected %d URLs, got %d", expectedCount, store.Count())
	}

	// Test lecturas concurrentes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				shortCode := fmt.Sprintf("code%d_%d", id, j)
				expectedURL := fmt.Sprintf("https://example.com/%d/%d", id, j)

				if url, exists := store.Get(shortCode); !exists || url != expectedURL {
					t.Errorf("Expected URL %s for code %s, got %s (exists: %v)",
						expectedURL, shortCode, url, exists)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestService_ShortenURL(t *testing.T) {
	store := NewStore()
	service := NewService(store)

	tests := []struct {
		name        string
		longURL     string
		expectError bool
		errorType   error
	}{
		{
			name:        "URL válida",
			longURL:     "https://www.example.com/path?param=value",
			expectError: false,
		},
		{
			name:        "URL con HTTP",
			longURL:     "http://example.com",
			expectError: false,
		},
		{
			name:        "URL vacía",
			longURL:     "",
			expectError: true,
			errorType:   ErrEmptyURL,
		},
		{
			name:        "URL solo espacios",
			longURL:     "   ",
			expectError: true,
			errorType:   ErrEmptyURL,
		},
		{
			name:        "URL sin esquema",
			longURL:     "www.example.com",
			expectError: true,
			errorType:   ErrInvalidURL,
		},
		{
			name:        "URL inválida",
			longURL:     "not-a-url",
			expectError: true,
			errorType:   ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortCode, err := service.ShortenURL(tt.longURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if tt.errorType != nil && err != tt.errorType {
					t.Errorf("Expected error %v, got %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(shortCode) != ShortCodeLength {
					t.Errorf("Expected short code length %d, got %d", ShortCodeLength, len(shortCode))
				}

				// Verificar que el código se guardó correctamente
				retrievedURL, err := service.GetLongURL(shortCode)
				if err != nil {
					t.Errorf("Error retrieving URL: %v", err)
				}
				if retrievedURL != tt.longURL {
					t.Errorf("Expected URL %s, got %s", tt.longURL, retrievedURL)
				}
			}
		})
	}
}

func TestService_GetLongURL(t *testing.T) {
	store := NewStore()
	service := NewService(store)

	// Agregar una URL de prueba
	testURL := "https://www.example.com"
	shortCode, err := service.ShortenURL(testURL)
	if err != nil {
		t.Fatalf("Error creating short URL: %v", err)
	}

	// Test obtener URL existente
	retrievedURL, err := service.GetLongURL(shortCode)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if retrievedURL != testURL {
		t.Errorf("Expected URL %s, got %s", testURL, retrievedURL)
	}

	// Test obtener URL no existente
	_, err = service.GetLongURL("nonexistent")
	if err != ErrURLNotFound {
		t.Errorf("Expected ErrURLNotFound, got %v", err)
	}
}

func TestService_UniqueCodeGeneration(t *testing.T) {
	store := NewStore()
	service := NewService(store)

	// Generar múltiples códigos para la misma URL
	testURL := "https://www.example.com"
	codes := make(map[string]bool)

	for i := 0; i < 100; i++ {
		shortCode, err := service.ShortenURL(testURL)
		if err != nil {
			t.Errorf("Error generating short code: %v", err)
		}

		if codes[shortCode] {
			t.Errorf("Duplicate short code generated: %s", shortCode)
		}
		codes[shortCode] = true
	}
}

func TestService_CollisionResistance(t *testing.T) {
	store := NewStore()
	service := NewService(store)

	// Llenar el store con códigos para forzar colisiones
	for i := 0; i < 1000; i++ {
		testURL := fmt.Sprintf("https://example%d.com", i)
		_, err := service.ShortenURL(testURL)
		if err != nil {
			t.Errorf("Error generating short code %d: %v", i, err)
		}
	}

	// Verificar que aún puede generar códigos únicos
	newURL := "https://newexample.com"
	shortCode, err := service.ShortenURL(newURL)
	if err != nil {
		t.Errorf("Error generating short code after many insertions: %v", err)
	}

	// Verificar que el código es único
	retrievedURL, err := service.GetLongURL(shortCode)
	if err != nil {
		t.Errorf("Error retrieving URL: %v", err)
	}
	if retrievedURL != newURL {
		t.Errorf("Expected URL %s, got %s", newURL, retrievedURL)
	}
}

func TestService_ConcurrentAccess(t *testing.T) {
	store := NewStore()
	service := NewService(store)

	const numGoroutines = 100
	const urlsPerGoroutine = 10

	// Defer para cleanup y logging siguiendo la Guía 2
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic en test concurrente: %v", r)
		}
	}()

	var wg sync.WaitGroup
	results := make(chan string, numGoroutines*urlsPerGoroutine)
	errors := make(chan error, numGoroutines*urlsPerGoroutine)

	// Lanzar múltiples goroutines con control de flujo mejorado
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			// Defer para cleanup de goroutine
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("panic en goroutine %d: %v", goroutineID, r)
				}
			}()

			// Loop interno con etiqueta para control de flujo
		urlLoop:
			for j := 0; j < urlsPerGoroutine; j++ {
				testURL := fmt.Sprintf("https://example%d-%d.com", goroutineID, j)
				
				// Switch para diferentes estrategias según el índice
				switch {
				case j < 3:
					// URLs normales
					if shortCode, err := service.ShortenURL(testURL); err != nil {
						errors <- fmt.Errorf("error en goroutine %d, URL %d: %v", goroutineID, j, err)
						break urlLoop // Salir del loop interno
					} else {
						results <- shortCode
					}
				case j < 7:
					// URLs con parámetros
					testURLWithParams := fmt.Sprintf("%s?param=%d", testURL, j)
					if shortCode, err := service.ShortenURL(testURLWithParams); err != nil {
						errors <- fmt.Errorf("error en goroutine %d, URL con params %d: %v", goroutineID, j, err)
						continue urlLoop // Continuar con la siguiente URL
					} else {
						results <- shortCode
					}
				default:
					// URLs complejas
					complexURL := fmt.Sprintf("%s/path/to/resource?param1=%d&param2=value", testURL, j)
					if shortCode, err := service.ShortenURL(complexURL); err != nil {
						errors <- fmt.Errorf("error en goroutine %d, URL compleja %d: %v", goroutineID, j, err)
						return // Salir de la goroutine si hay error crítico
					} else {
						results <- shortCode
					}
				}
			}
		}(i)
	}

	// Esperar a que terminen todas las goroutines
	wg.Wait()
	close(results)
	close(errors)

	// Verificar errores con range idiomático
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}

	// Verificar unicidad de códigos con range idiomático
	codes := make(map[string]bool)
	for code := range results {
		if codes[code] {
			t.Errorf("Duplicate code generated: %s", code)
		}
		codes[code] = true
	}

	expectedCount := numGoroutines * urlsPerGoroutine
	if len(codes) != expectedCount {
		t.Errorf("Expected %d unique codes, got %d", expectedCount, len(codes))
	}
}

func BenchmarkService_ShortenURL(b *testing.B) {
	store := NewStore()
	service := NewService(store)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testURL := fmt.Sprintf("https://benchmark%d.com", i)
		_, err := service.ShortenURL(testURL)
		if err != nil {
			b.Errorf("Error in benchmark: %v", err)
		}
	}
}

func BenchmarkService_GetLongURL(b *testing.B) {
	store := NewStore()
	service := NewService(store)

	// Preparar datos de prueba
	testCodes := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		testURL := fmt.Sprintf("https://benchmark%d.com", i)
		shortCode, _ := service.ShortenURL(testURL)
		testCodes[i] = shortCode
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code := testCodes[i%len(testCodes)]
		_, err := service.GetLongURL(code)
		if err != nil {
			b.Errorf("Error in benchmark: %v", err)
		}
	}
}
