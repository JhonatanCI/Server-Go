package main

import (
	"net/http"
	"time" // Nuevo import

	"github.com/golang-jwt/jwt/v5" // Nuevo import
	"github.com/labstack/echo-jwt/v4" // Nuevo import
	"github.com/labstack/echo/v4"
	"github.com/jackc/pgx/v5/pgxpool" 
	"context"
	"log"
)

// --- CONSTANTES ---
// Esta es la clave que nuestro agente debe presentar para obtener un token.
const claveSecretaAgente = "un-secreto-muy-secreto-para-los-agentes"
// Esta es la clave que el servidor usará para FIRMAR los tokens. Solo el servidor la conoce.
const claveSecretaJWT = "un-secreto-aun-mas-secreto-para-firmar"

type Metricas struct {
	UsoCPU   float64 `json:"uso_cpu"`
	UsoDisco float64 `json:"uso_disco"`
}

type ServidorAPI struct {
	db *pgxpool.Pool
}

type LoginRequest struct {
	ClaveSecreta string `json:"clave_secreta_agente"`
}

func holaMundoHandler(c echo.Context) error {
	return c.String(http.StatusOK, "¡El servidor central está en línea!")
}

// Handler para recibir las métricas del agente
// main.go

// recibirMetricasHandler ahora es un método de ServidorAPI
func (s *ServidorAPI) recibirMetricasHandler(c echo.Context) error {
	// --- Esta parte es la misma de antes ---
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	var m Metricas
	if err := c.Bind(&m); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "json invalido"})
	}
	log.Printf("Métricas recibidas: %+v. Token del agente expira en: %v", m, claims["exp"])
	// ------------------------------------

	// Por ahora, asumimos que todas las métricas vienen del servidor con ID=1.
	servidorID := 1
	
	// La consulta SQL para insertar los datos. ¡Usamos $1, $2, $3 para prevenir inyección SQL!
	query := "INSERT INTO metricas (servidor_id, uso_cpu, uso_disco) VALUES ($1, $2, $3)"

	// Ejecutamos la consulta usando la conexión del servidor 's.db'
	_, err := s.db.Exec(context.Background(), query, servidorID, m.UsoCPU, m.UsoDisco)
	if err != nil {
		log.Printf("Error al insertar métricas en la base de datos: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "no se pudieron guardar las métricas"})
	}

	log.Printf("Métricas del servidor %d guardadas exitosamente.", servidorID)
	// -----------------------------------------

	// --- Respuesta modificada ---
	return c.JSON(http.StatusOK, map[string]string{"status": "métricas recibidas y guardadas"})
}

// main.go

// getMetricasHandler devuelve las últimas N métricas de un servidor.
func (s *ServidorAPI) getMetricasHandler(c echo.Context) error {
	// Obtenemos el ID de la URL, ej. /api/servers/1/metrics -> id = "1"
	id := c.Param("id")

	// Consulta compleja que une las dos tablas.
	query := `
		SELECT m.uso_cpu, m.uso_disco, m.recolectado_en, s.nombre
		FROM metricas m
		INNER JOIN servidores s ON m.servidor_id = s.id
		WHERE s.id = $1
		ORDER BY m.recolectado_en DESC
		LIMIT 10
	`

	rows, err := s.db.Query(context.Background(), query, id)
	if err != nil {
		log.Printf("Error al consultar métricas: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "error en la base de datos"})
	}
	defer rows.Close()

	// Creamos un slice para guardar los resultados.
	var resultados []map[string]interface{}
	for rows.Next() {
		var usoCPU, usoDisco float64
		var recolectadoEn time.Time
		var nombreServidor string

		if err := rows.Scan(&usoCPU, &usoDisco, &recolectadoEn, &nombreServidor); err != nil {
			log.Printf("Error al escanear fila: %v", err)
			continue // Salta esta fila si hay un error
		}
		
		resultado := map[string]interface{}{
			"servidor":       nombreServidor,
			"uso_cpu":        usoCPU,
			"uso_disco":      usoDisco,
			"recolectado_en": recolectadoEn,
		}
		resultados = append(resultados, resultado)
	}

	return c.JSON(http.StatusOK, resultados)
}

func loginHandler(c echo.Context) error {
	// 1. Vinculamos el JSON del body a nuestro struct LoginRequest
	req := new(LoginRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "json invalido"})
	}

	// 2. Verificamos si la clave secreta del agente es correcta
	if req.ClaveSecreta != claveSecretaAgente {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "clave de agente invalida"})
	}

	// 3. Si la clave es correcta, creamos el token JWT
	claims := &jwt.RegisteredClaims{
		// Podemos añadir información extra aquí si quisiéramos (ej. nombre del agente)
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)), // El token expira en 3 días
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 4. Firmamos el token con nuestra clave secreta JWT y lo enviamos al agente
	t, err := token.SignedString([]byte(claveSecretaJWT))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{
		"token": t,
	})
}

// main.go

func main() {
    // --- NUEVO: Conexión a la Base de Datos ---
	// URL de conexión a tu base de datos.
	// Formato: postgres://USUARIO:CONTRASEÑA@HOST:PUERTO/NOMBRE_DB
	dbURL := "postgres://monitor_user:tu_contraseña_segura@localhost:5432/monitor_db"

	// Creamos el pool de conexiones.
	dbpool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("No se pudo conectar a la base de datos: %v", err)
	}
	defer dbpool.Close()

	// Verificamos que la conexión esté viva.
	if err := dbpool.Ping(context.Background()); err != nil {
		log.Fatalf("No se pudo hacer ping a la base de datos: %v", err)
	}
	
	log.Println("Conexión a la base de datos establecida exitosamente.")

	// Creamos nuestra instancia del servidor con la conexión a la BD.
	api := &ServidorAPI{
		db: dbpool,
	}
    // --- FIN DE LA SECCIÓN NUEVA ---

	e := echo.New()

	// --- RUTAS ---
	e.GET("/", holaMundoHandler) // Esta sigue siendo una función normal
	e.POST("/login", loginHandler) // Esta también

	g := e.Group("/api")
	config := echojwt.Config{
		SigningKey: []byte(claveSecretaJWT),
	}
	g.Use(echojwt.WithConfig(config))
	
	// --- RUTAS MODIFICADAS ---
	// La ruta ahora llama a los MÉTODOS de nuestra instancia 'api'.
	g.POST("/metrics", api.recibirMetricasHandler)
    // --- NUEVA RUTA ---
    g.GET("/servers/:id/metrics", api.getMetricasHandler)


	e.Logger.Fatal(e.Start(":8080"))
}