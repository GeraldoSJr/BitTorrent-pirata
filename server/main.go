package main

import (
	"bufio"
	"fmt"
	"net"
	"github.com/GeraldoSJr/BitTorrent-pirata/v2/helpers"
)

type Server struct {
	storage *helpers.IPStorage
}

func NewServer() *Server {
	return &Server{
		storage: &helpers.IPStorage{
			Data: make(map[string]helpers.FileInfo),
		},
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		clientAddr := conn.RemoteAddr().String()
		fmt.Printf("Client disconnected: %s\n", clientAddr)
		s.storage.RemoveClient(clientAddr)
		conn.Close()
	}()

	clientAddr := conn.RemoteAddr().String()
	fmt.Printf("Client connected: %s\n", clientAddr)

	reader := bufio.NewReader(conn)
	fmt.Fprintf(conn, "Send FileInfo as a JSON object:\n")
	message, _ := reader.ReadString('\n')

	fileInfo, err := helpers.DecodeFileInfo([]byte(message))
	if err != nil {
		fmt.Fprintf(conn, "Invalid JSON format: %v\n", err)
		return
	}

	s.storage.AddClientInfo(clientAddr, fileInfo)

	clients := s.storage.GetAllClients()
	conn.Write([]byte("List of clients and file information:\n"))
	for ip, info := range clients {
		conn.Write([]byte(fmt.Sprintf("IP: %s, File Hash: %s, File ID: %s\n", ip, info.FileHash, info.FileID)))
	}
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
		go s.handleClient(conn)
	}
}

func main() {
	server := NewServer()
	server.listen(":8080")
}
