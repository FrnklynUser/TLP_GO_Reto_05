package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"acortador-urls/internal/shortener"
)

// Handler maneja las peticiones HTTP
type Handler struct {
	service *shortener.Service
}

// NewHandler crea una nueva instancia del handler
func NewHandler(service *shortener.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// ShortenRequest representa la petición para acortar una URL con validaciones
type ShortenRequest struct {
	LongURL string `json:"long_url" validate:"required,url" example:"https://www.example.com"`
}

// ShortenResponse representa la respuesta con la URL acortada
type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}



// ShortenURL maneja las peticiones POST /shorten con validación temprana
func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	// Configurar headers de respuesta
	w.Header().Set("Content-Type", "application/json")

	// Validación temprana: verificar Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		h.sendErrorResponse(w, http.StatusBadRequest, "invalid_content_type", "Content-Type debe ser application/json")
		return
	}

	// Validación temprana: verificar método HTTP
	if r.Method != http.MethodPost {
		h.sendErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Método no permitido")
		return
	}

	// Decodificar el cuerpo de la petición
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "invalid_json", fmt.Sprintf("Formato JSON inválido: %v", err))
		return
	}

	// Validación temprana: verificar que la URL no esté vacía (redundante pero defensiva)
	if strings.TrimSpace(req.LongURL) == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "empty_url", "La URL no puede estar vacía")
		return
	}

	// Defer para logging de requests siguiendo la Guía 2
	defer func() {
		if r := recover(); r != nil {
			h.sendErrorResponse(w, http.StatusInternalServerError, "panic_error", fmt.Sprintf("Error crítico: %v", r))
		}
	}()

	// Acortar la URL con manejo idiomático de errores
	if shortCode, err := h.service.ShortenURL(req.LongURL); err != nil {
		// Switch idiomático para diferentes tipos de error
		switch {
		case errors.Is(err, shortener.ErrInvalidURL):
			h.sendErrorResponse(w, http.StatusBadRequest, "invalid_url", "URL inválida")
		case errors.Is(err, shortener.ErrEmptyURL):
			h.sendErrorResponse(w, http.StatusBadRequest, "empty_url", "La URL no puede estar vacía")
		case errors.Is(err, shortener.ErrMaxRetries):
			h.sendErrorResponse(w, http.StatusInternalServerError, "generation_failed", "No se pudo generar un código único")
		case strings.Contains(err.Error(), "crítico"):
			h.sendErrorResponse(w, http.StatusInternalServerError, "critical_error", "Error crítico del sistema")
		default:
			h.sendErrorResponse(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Error interno: %v", err))
		}
		return
	} else {
		// Construir la URL corta completa solo si fue exitoso
		baseURL := h.getBaseURL(r)
		shortURL := fmt.Sprintf("%s/%s", baseURL, shortCode)

		// Enviar respuesta exitosa
		response := ShortenResponse{
			ShortURL: shortURL,
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// RedirectURL maneja las peticiones GET /{short_code} con patrones idiomáticos de Go
func (h *Handler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	// Defer para logging y panic recovery siguiendo la Guía 2
	defer func() {
		if r := recover(); r != nil {
			h.sendErrorResponse(w, http.StatusInternalServerError, "panic_error", fmt.Sprintf("Error crítico en redirección: %v", r))
		}
	}()

	// Obtener y validar el código corto con if idiomático
	if shortCode := chi.URLParam(r, "short_code"); shortCode == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "missing_code", "Código corto requerido")
		return
	} else {
		// Buscar la URL larga con manejo idiomático de errores
		if longURL, err := h.service.GetLongURL(shortCode); err != nil {
			// Switch idiomático para diferentes tipos de error
			switch {
			case errors.Is(err, shortener.ErrURLNotFound):
				h.sendErrorResponse(w, http.StatusNotFound, "not_found", "Código corto no encontrado")
			case strings.Contains(err.Error(), "crítico"):
				h.sendErrorResponse(w, http.StatusInternalServerError, "critical_error", "Error crítico del sistema")
			default:
				h.sendErrorResponse(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Error interno: %v", err))
			}
			return
		} else {
			// Redirigir a la URL larga usando HTTP 307 (Temporary Redirect)
			// Justificación: HTTP 307 preserva el método HTTP original y es más apropiado
			// para redirecciones temporales que pueden cambiar en el futuro
			w.Header().Set("Location", longURL)
			w.WriteHeader(http.StatusTemporaryRedirect)
		}
	}
}

// getBaseURL construye la URL base del servidor
func (h *Handler) getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// Verificar headers de proxy
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}

	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}

// sendErrorResponse envía una respuesta de error en formato JSON
func (h *Handler) sendErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errorResponse := ErrorResponse{
		Error:   errorCode,
		Message: message,
	}
	json.NewEncoder(w).Encode(errorResponse)
}
