# Acortador de URLs Inteligente y Resistente

Un servicio HTTP en Go para generar URLs cortas que sean fáciles de compartir y recordar, especialmente para campañas de marketing o enlaces de documentación interna.

## Características

- **API HTTP RESTful** con endpoints para acortar y redirigir URLs
- **Generación inteligente de códigos cortos** resistente a colisiones
- **Almacenamiento concurrente seguro** usando sync.RWMutex
- **Validación exhaustiva** de URLs de entrada
- **Manejo robusto de errores** con códigos HTTP semánticos
- **Arquitectura modular** con separación clara de responsabilidades

## Estructura del Proyecto

```
acortador-urls/
├── cmd/api/
│   └── main.go                 # Punto de entrada del servidor
├── internal/
│   ├── handlers/
│   │   ├── http.go            # Manejadores HTTP
│   │   └── http_test.go       # Pruebas de integración
│   └── shortener/
│       ├── service.go         # Lógica de negocio
│       ├── store.go           # Almacenamiento concurrente
│       └── shortener_test.go  # Pruebas unitarias
├── go.mod                     # Dependencias del módulo
└── README.md                  # Documentación
```

### Justificación de la Estructura

- **`cmd/api/`**: Contiene el punto de entrada del servidor, siguiendo las convenciones de Go para aplicaciones ejecutables
- **`internal/handlers/`**: Maneja las peticiones HTTP y las respuestas, separando la lógica de presentación
- **`internal/shortener/`**: Contiene la lógica de negocio central (generación de códigos, validación, almacenamiento)
- **Separación de responsabilidades**: Cada paquete tiene una responsabilidad específica y bien definida

## API Endpoints

### POST /shorten
Acorta una URL larga y retorna el código corto generado.

**Request:**
```json
{
  "long_url": "https://muy-larga-url-de-ejemplo.com/path/to/resource?param=value"
}
```

**Response (201 Created):**
```json
{
  "short_url": "http://localhost:8080/abc12d"
}
```

**Errores:**
- `400 Bad Request`: URL inválida o vacía
- `500 Internal Server Error`: Error al generar código único

### GET /{short_code}
Redirige a la URL larga asociada con el código corto.

**Response:**
- `307 Temporary Redirect`: Redirige a la URL larga
- `404 Not Found`: Código corto no encontrado
- `400 Bad Request`: Código corto vacío

## Algoritmo de Generación de Códigos Cortos

### Estrategia de Generación

El algoritmo combina múltiples fuentes de entropía para generar códigos únicos:

1. **Timestamp actual** (`time.Now().UnixNano()`)
2. **Número aleatorio** (`rand.Int63()`)
3. **URL original** (como entrada adicional)
4. **Número de intento** (para manejo de colisiones)

### Proceso de Generación

```go
baseString := fmt.Sprintf("%s-%d-%d-%d", longURL, timestamp, randomNum, attempt)
hash := md5.Sum([]byte(baseString))
shortCode := extractValidChars(hex.EncodeToString(hash), 6)
```

### Manejo de Colisiones

- **Reintentos automáticos**: Hasta 10 intentos para generar un código único
- **Incremento del número de intento**: Cada reintento modifica la entrada del hash
- **Verificación de unicidad**: Cada código se verifica contra el almacén antes de ser aceptado
- **Prevención de bucles infinitos**: Límite máximo de reintentos para evitar bloqueos

### Características del Código Generado

- **Longitud fija**: 6 caracteres alfanuméricos
- **Caracteres válidos**: `a-z`, `A-Z`, `0-9` (62 caracteres posibles)
- **Espacio de códigos**: 62^6 = ~56.8 billones de combinaciones posibles

## Elección de Redirección HTTP: 307 vs 301

### Decisión: HTTP 307 Temporary Redirect

**Justificación:**

1. **Preservación del método HTTP**: 307 garantiza que el método original se mantenga
2. **No cacheo permanente**: A diferencia de 301, los navegadores no cachean permanentemente la redirección
3. **Flexibilidad**: Permite cambios futuros en las URLs de destino
4. **Comportamiento predecible**: Evita problemas con clientes que podrían manejar mal las redirecciones permanentes

**Comparación:**
- **301 Moved Permanently**: Indica que la URL se ha movido permanentemente, los navegadores pueden cachear indefinidamente
- **307 Temporary Redirect**: Indica una redirección temporal, preserva el método HTTP original

Para un acortador de URLs, 307 es más apropiado porque:
- Las URLs cortas pueden ser reutilizadas o modificadas
- No queremos que los navegadores asuman que la redirección es permanente
- Mantenemos control total sobre el comportamiento de redirección

## Concurrencia y Seguridad

### Almacenamiento Concurrente

El almacenamiento utiliza `sync.RWMutex` para garantizar acceso seguro:

```go
type Store struct {
    urls map[string]string
    mu   sync.RWMutex
}
```

### Operaciones Thread-Safe

- **Escritura** (`Save`): Usa `mu.Lock()` para acceso exclusivo
- **Lectura** (`Get`, `Exists`): Usa `mu.RLock()` para acceso compartido
- **Conteo** (`Count`): Usa `mu.RLock()` para lectura segura

### ¿Por qué sync.RWMutex?

Un `map[string]string` simple no es seguro para concurrencia en Go porque:
- Las operaciones de escritura pueden corromper la estructura interna del mapa
- Las lecturas concurrentes con escrituras pueden causar panics
- Go detecta estas condiciones de carrera y termina el programa

`sync.RWMutex` resuelve esto permitiendo:
- **Múltiples lectores simultáneos** cuando no hay escritores
- **Un solo escritor exclusivo** cuando se necesita modificar el mapa
- **Prevención de condiciones de carrera** mediante sincronización explícita

## Instalación y Uso

### Prerrequisitos
- Go 1.21 o superior

### Instalación

```bash
# Clonar el repositorio
git clone <repository-url>
cd acortador-urls

# Descargar dependencias
go mod tidy

# Ejecutar pruebas
go test ./... -race

# Ejecutar el servidor
go run cmd/api/main.go
```

### Uso

El servidor se ejecuta por defecto en el puerto 8080:

```bash
# Acortar una URL
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"long_url": "https://www.example.com/very/long/path"}'

# Usar la URL corta (redirige automáticamente)
curl -L http://localhost:8080/abc12d
```

## Pruebas

### Ejecutar Todas las Pruebas

```bash
# Pruebas unitarias y de integración
go test ./...

# Pruebas con detección de condiciones de carrera
go test ./... -race

# Pruebas con cobertura
go test ./... -cover

# Benchmarks
go test ./... -bench=.
```

### Tipos de Pruebas Incluidas

1. **Pruebas Unitarias**:
   - Generación de códigos cortos
   - Validación de URLs
   - Almacenamiento concurrente
   - Manejo de colisiones

2. **Pruebas de Integración**:
   - Endpoints HTTP completos
   - Flujo completo acortar → redirigir
   - Manejo de errores HTTP

3. **Pruebas de Concurrencia**:
   - Acceso concurrente al almacén
   - Generación simultánea de códigos
   - Peticiones HTTP concurrentes

4. **Benchmarks**:
   - Rendimiento de generación de códigos
   - Rendimiento de endpoints HTTP
   - Rendimiento de operaciones de almacenamiento

## Configuración

### Variables de Entorno

- `PORT`: Puerto del servidor (default: 8080)

### Ejemplo

```bash
PORT=3000 go run cmd/api/main.go
```

## Limitaciones y Consideraciones

1. **Almacenamiento en memoria**: Los datos se pierden al reiniciar el servidor
2. **Escalabilidad**: Limitado por la memoria disponible del servidor
3. **Persistencia**: No hay persistencia real, solo simulada en memoria
4. **Distribución**: No está diseñado para múltiples instancias

## Posibles Mejoras Futuras

1. **Persistencia real**: Integración con bases de datos (PostgreSQL, Redis)
2. **Métricas**: Contadores de clicks y estadísticas de uso
3. **Expiración**: URLs con tiempo de vida limitado
4. **API de gestión**: Endpoints para listar, modificar o eliminar URLs
5. **Autenticación**: Control de acceso para crear URLs cortas
6. **Rate limiting**: Protección contra abuso del servicio
