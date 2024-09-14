package main

import (
	"fmt"
	"net"
	"sync"
)

type Server struct {
	clients map[string]net.Conn
	mu      sync.Mutex
}

func NewServer() *Server {
	return &Server{
		clients: make(map[string]net.Conn),
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	fmt.Printf("Client connected: %s\n", clientAddr)

	// Add client to the list
	s.mu.Lock()
	s.clients[clientAddr] = conn
	s.mu.Unlock()

	// Listen for client messages
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("Client disconnected: %s\n", clientAddr)
			break
		}

		msg := string(buffer[:n])
		fmt.Printf("Message from %s: %s\n", clientAddr, msg)
	}

	// Remove client after disconnection
	s.mu.Lock()
	delete(s.clients, clientAddr)
	s.mu.Unlock()
}

func (s *Server) listen(port string) {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started on port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go s.handleClient(conn) // Handle client concurrently
	}
}

func main() {
	server := NewServer()
	server.listen(":8080")
}
