package shortener

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"
)

// Configuración del servicio de acortador
const (
	// ShortCodeLength define la longitud fija del código corto generado
	ShortCodeLength = 6
	// MaxRetries define el máximo número de reintentos para evitar colisiones
	MaxRetries = 10
	// ValidChars contiene todos los caracteres válidos para el código corto
	ValidChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// Estados de validación usando iota
type ValidationResult int

const (
	ValidationSuccess ValidationResult = iota
	ValidationErrorEmpty
	ValidationErrorInvalid
	ValidationErrorMalformed
)

// Errores predefinidos del servicio siguiendo mejores prácticas
var (
	ErrInvalidURL     = errors.New("URL inválida")
	ErrEmptyURL       = errors.New("URL no puede estar vacía")
	ErrMaxRetries     = errors.New("máximo número de reintentos alcanzado para generar código único")
	ErrURLNotFound    = errors.New("URL no encontrada")
	ErrServiceUnavailable = errors.New("servicio no disponible")
)

// ValidationError representa un error de validación con contexto
type ValidationError struct {
	Field string
	Value interface{}
	Msg   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validación falló en campo '%s' con valor '%v': %s", e.Field, e.Value, e.Msg)
}

// Service contiene la lógica de negocio del acortador
type Service struct {
	store *Store
	rand  *rand.Rand
}

// NewService crea una nueva instancia del servicio
func NewService(store *Store) *Service {
	return &Service{
		store: store,
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ShortenURL acorta una URL larga y retorna el código corto usando patrones idiomáticos de Go
func (s *Service) ShortenURL(longURL string) (shortCode string, err error) {
	// Defer para logging y cleanup siguiendo la Guía 2
	defer func() {
		if r := recover(); r != nil {
			// Recover de panic crítico
			err = fmt.Errorf("error crítico en ShortenURL: %v", r)
			shortCode = ""
		}
	}()

	// Validación temprana con if idiomático
	if err := s.validateURL(longURL); err != nil {
		return "", err
	}

	// Generar código corto único con manejo robusto
	if shortCode, err := s.generateUniqueShortCode(longURL); err != nil {
		return "", err
	} else {
		// Almacenar la relación solo si la generación fue exitosa
		s.store.Save(shortCode, longURL)
		return shortCode, nil
	}
}

// GetLongURL obtiene la URL larga asociada a un código corto con patrones idiomáticos
func (s *Service) GetLongURL(shortCode string) (longURL string, err error) {
	// Defer para logging y cleanup siguiendo la Guía 2
	defer func() {
		if r := recover(); r != nil {
			// Recover de panic crítico
			err = fmt.Errorf("error crítico en GetLongURL: %v", r)
			longURL = ""
		}
	}()

	// Validación temprana con if idiomático
	if trimmedCode := strings.TrimSpace(shortCode); trimmedCode == "" {
		return "", ErrEmptyURL
	} else {
		// Buscar en el almacén con manejo idiomático
		if longURL, exists := s.store.Get(trimmedCode); !exists {
			return "", ErrURLNotFound
		} else {
			return longURL, nil
		}
	}
}

// validateURL valida que la URL sea válida usando named return values y validaciones múltiples
func (s *Service) validateURL(longURL string) (err error) {
	// Validaciones múltiples usando funciones variádicas
	if err = s.validateURLBasics(longURL); err != nil {
		return err
	}

	if err = s.validateURLFormat(longURL); err != nil {
		return err
	}

	if err = s.validateURLSecurity(longURL); err != nil {
		return err
	}

	return nil // Named return value
}

// validateURLBasics realiza validaciones básicas
func (s *Service) validateURLBasics(longURL string) error {
	if longURL == "" {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "no puede estar vacía"}
	}

	if strings.TrimSpace(longURL) == "" {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "no puede contener solo espacios"}
	}

	return nil
}

// validateURLFormat valida el formato de la URL
func (s *Service) validateURLFormat(longURL string) error {
	parsedURL, err := url.Parse(longURL)
	if err != nil {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "formato inválido"}
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "debe usar esquema http o https"}
	}

	if parsedURL.Host == "" {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "debe tener un host válido"}
	}

	return nil
}

// validateURLSecurity realiza validaciones de seguridad
func (s *Service) validateURLSecurity(longURL string) error {
	// Lista de dominios bloqueados (ejemplo de validación de seguridad)
	blockedDomains := []string{"malware.com", "phishing.net", "spam.org"}
	
	parsedURL, _ := url.Parse(longURL)
	for _, blocked := range blockedDomains {
		if strings.Contains(parsedURL.Host, blocked) {
			return &ValidationError{Field: "long_url", Value: longURL, Msg: "dominio bloqueado por seguridad"}
		}
	}

	return nil
}

// generateUniqueShortCode genera un código corto único resistente a colisiones con retry pattern
func (s *Service) generateUniqueShortCode(longURL string) (string, error) {
	// Defer para logging de intentos siguiendo la Guía 2
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("error crítico en generación de código: %v", r))
		}
	}()

	// Retry pattern con for loop idiomático
	for attempt := 0; attempt < MaxRetries; attempt++ {
		// Switch para manejar diferentes estrategias según el intento
		var shortCode string
		switch {
		case attempt < 3:
			// Primeros intentos: estrategia normal
			shortCode = s.generateShortCode(longURL, attempt)
		case attempt < 7:
			// Intentos intermedios: agregar más entropía
			shortCode = s.generateShortCode(longURL, attempt*2) // Más variación
		default:
			// Últimos intentos: estrategia agresiva con timestamp
			shortCode = s.generateShortCode(longURL+fmt.Sprintf("_%d", time.Now().UnixNano()), attempt)
		}
		
		// Verificar si el código ya existe
		if !s.store.Exists(shortCode) {
			return shortCode, nil
		}
	}
	
	return "", ErrMaxRetries
}

// generateShortCode genera un código corto usando closure para entrada única
func (s *Service) generateShortCode(longURL string, attempt int) string {
	// Usar closure para generar entrada única
	entryGenerator := s.createEntryGenerator(longURL, attempt)
	entry := entryGenerator()
	
	// Generar hash MD5
	hash := md5.Sum([]byte(entry))
	hashString := hex.EncodeToString(hash[:])
	
	// Tomar los primeros caracteres y convertir a base alfanumérica
	result := make([]byte, ShortCodeLength)
	for i := 0; i < ShortCodeLength; i++ {
		index := int(hashString[i]) % len(ValidChars)
		result[i] = ValidChars[index]
	}
	
	return string(result)
}

// createEntryGenerator crea un closure para generar entradas únicas
func (s *Service) createEntryGenerator(longURL string, attempt int) func() string {
	// Variables capturadas por el closure
	timestamp := time.Now().UnixNano()
	randomValue := s.rand.Int63()
	
	return func() string {
		var builder strings.Builder
		builder.Grow(len(longURL) + 50) // Pre-allocar para mejor performance
		
		builder.WriteString(longURL)
		builder.WriteString("_")
		builder.WriteString(fmt.Sprintf("%d", timestamp))
		builder.WriteString("_")
		builder.WriteString(fmt.Sprintf("%d", attempt))
		builder.WriteString("_")
		builder.WriteString(fmt.Sprintf("%d", randomValue))
		
		return builder.String()
	}
}

// GetStats retorna estadísticas del servicio
func (s *Service) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_urls": s.store.Count(),
	}
}
