// en client.go
package main

import (
	"log"
	"golang.org/x/net/websocket" // Necesitaremos esta librería
)

// Client es un intermediario entre la conexión websocket y el hub.
type Client struct {
	hub *Hub
	// La conexión websocket en sí.
	conn *websocket.Conn
	// Canal de salida de mensajes.
	send chan []byte
}

// readPump bombea mensajes desde la conexión websocket hacia el hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		// Leemos un mensaje de la conexión. Si hay un error, asumimos que el cliente se desconectó.
		var msg = make([]byte, 1024)
		if _, err := c.conn.Read(msg); err != nil {
			log.Printf("error en readPump: %v", err)
			break
		}
	}
}

// writePump bombea mensajes desde el hub hacia la conexión websocket.
func (c *Client) writePump() {
	defer c.conn.Close()
	for message := range c.send {
		// Escribimos el mensaje en la conexión.
		if err := websocket.Message.Send(c.conn, message); err != nil {
			log.Printf("error en writePump: %v", err)
			return
		}
	}
}