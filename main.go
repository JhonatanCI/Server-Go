package main

import (
	"net/http"
	"github.com/labstack/echo/v4"
)

func holaMundoHandler(c echo.Context) error {

	return c.String(http.StatusOK, "¡El servidor central está en línea!")
}

func main() {
	e := echo.New()

	e.GET("/", holaMundoHandler)

	e.Logger.Fatal(e.Start(":8080"))
}