// en hub.go
package main

import "log"

// Hub mantiene el conjunto de clientes activos y transmite mensajes a los clientes.
type Hub struct {
    // Clientes registrados. El `bool` es solo para que funcione como un `set`.
    clients map[*Client]bool

    // Mensajes entrantes de los clientes para transmitir.
    broadcast chan []byte

    // Canal para registrar solicitudes de nuevos clientes.
    register chan *Client

    // Canal para anular el registro de clientes.
    unregister chan *Client
}

func newHub() *Hub {
    return &Hub{
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
    }
}

// Run es el bucle principal del Hub que maneja los canales.	
func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
            log.Println("Nuevo cliente conectado al Hub.")
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
                log.Println("Cliente desconectado del Hub.")
            }
        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}