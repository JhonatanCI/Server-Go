package main

import (
	"fmt"
	"net/http"
	"time" // Nuevo import

	"github.com/golang-jwt/jwt/v5" // Nuevo import
	"github.com/labstack/echo-jwt/v4" // Nuevo import
	"github.com/labstack/echo/v4"
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

type LoginRequest struct {
	ClaveSecreta string `json:"clave_secreta_agente"`
}

func holaMundoHandler(c echo.Context) error {
	return c.String(http.StatusOK, "¡El servidor central está en línea!")
}

// Handler para recibir las métricas del agente
func recibirMetricasHandler(c echo.Context) error {

	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*jwt.RegisteredClaims)
	
	var m Metricas


	if err := c.Bind(&m); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "json invalido"})
	}

	fmt.Printf("Métricas recibidas: %+v. Token expira en: %v\n", m, claims.ExpiresAt)

	return c.JSON(http.StatusOK, map[string]string{"status": "métricas recibidas"})
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

func main() {
	e := echo.New()

	// --- RUTAS PÚBLICAS ---
	e.GET("/", holaMundoHandler)
	e.POST("/login", loginHandler)

	// --- RUTAS PROTEGIDAS ---
	// Creamos un grupo de rutas que requerirán autenticación JWT.
	g := e.Group("/api")

	config := echojwt.Config{
		SigningKey:    []byte(claveSecretaJWT),
	}

	g.Use(echojwt.WithConfig(config))

	// Movemos nuestra ruta de métricas para que esté DENTRO del grupo protegido.
	g.POST("/metrics", recibirMetricasHandler)


	e.Logger.Fatal(e.Start(":8080"))
}