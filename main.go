package main

import (
	"fmt"
	"net/http"
	"github.com/labstack/echo/v4"
)

type Metricas struct {
	UsoCPU   float64 `json:"uso_cpu"`
	UsoDisco float64 `json:"uso_disco"`
}

func holaMundoHandler(c echo.Context) error {
	return c.String(http.StatusOK, "¡El servidor central está en línea!")
}

// Handler para recibir las métricas del agente
func recibirMetricasHandler(c echo.Context) error {
	
	var m Metricas


	if err := c.Bind(&m); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "json invalido"})
	}

	fmt.Printf("Métricas recibidas: %+v\n", m)

	return c.JSON(http.StatusOK, map[string]string{"status": "métricas recibidas"})
}

func main() {
	e := echo.New()

	e.GET("/", holaMundoHandler)

	e.POST("/api/metrics", recibirMetricasHandler)

	e.Logger.Fatal(e.Start(":8080"))
}