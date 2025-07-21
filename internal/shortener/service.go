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

// validateURL valida que la URL sea válida y no esté vacía usando validación temprana
func (s *Service) validateURL(longURL string) error {
	// Validación temprana: verificar string vacío
	if longURL == "" {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "no puede estar vacía"}
	}

	// Validación temprana: verificar solo espacios
	trimmedURL := strings.TrimSpace(longURL)
	if trimmedURL == "" {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "no puede contener solo espacios"}
	}

	// Validación temprana: longitud mínima razonable
	if len(trimmedURL) < 7 { // http:// mínimo
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "demasiado corta para ser una URL válida"}
	}

	// Validar formato de URL
	parsedURL, err := url.Parse(trimmedURL)
	if err != nil {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: fmt.Sprintf("formato inválido: %v", err)}
	}

	// Verificar que tenga esquema válido
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "debe usar esquema http o https"}
	}

	// Verificar que tenga host
	if parsedURL.Host == "" {
		return &ValidationError{Field: "long_url", Value: longURL, Msg: "debe tener un host válido"}
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

// generateShortCode genera un código corto usando tiempo, aleatoriedad y hash
func (s *Service) generateShortCode(longURL string, attempt int) string {
	// Combinar tiempo actual, URL y número de intento para mayor unicidad
	timestamp := time.Now().UnixNano()
	randomNum := s.rand.Int63()

	// Crear string base para el hash
	baseString := fmt.Sprintf("%s-%d-%d-%d", longURL, timestamp, randomNum, attempt)

	// Generar hash MD5
	hasher := md5.New()
	hasher.Write([]byte(baseString))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Extraer caracteres válidos del hash y crear código corto
	return s.extractValidChars(hash, ShortCodeLength)
}

// extractValidChars extrae caracteres alfanuméricos válidos del hash usando strings.Builder para mejor performance
func (s *Service) extractValidChars(hash string, length int) string {
	var result strings.Builder
	// Preasignar capacidad para evitar realocaciones (mejora de performance de la Guía 1)
	result.Grow(length)
	validCharsLen := len(ValidChars)
	
	// Extraer caracteres del hash primero
	for i := 0; i < len(hash) && result.Len() < length; i++ {
		// Usar el valor del byte para seleccionar un carácter válido
		charIndex := int(hash[i]) % validCharsLen
		result.WriteByte(ValidChars[charIndex])
	}
	
	// Si no tenemos suficientes caracteres, completar con aleatorios
	for result.Len() < length {
		charIndex := s.rand.Intn(validCharsLen)
		result.WriteByte(ValidChars[charIndex])
	}
	
	return result.String()
}

// GetStats retorna estadísticas del servicio
func (s *Service) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_urls": s.store.Count(),
	}
}
