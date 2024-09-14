package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
	"github.com/GeraldoSJr/BitTorrent-pirata/v2/helpers"
)

type Server struct {
	storage *helpers.IPStorage
}

func NewServer() *Server {
	return &Server{
		storage: helpers.NewIPStorage(),
	}
}

func (s *Server) monitorConnection(conn net.Conn, clientAddr string) {
	for {
		_, err := conn.Write([]byte(""))
		if err != nil {
			fmt.Printf("Client %s disconnected. Removing data...\n", clientAddr)
			s.storage.RemoveClient(clientAddr)
			conn.Close()
			return
		}
		time.Sleep(3 * time.Second)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		clientAddr := conn.RemoteAddr().String()
		fmt.Printf("Client disconnected: %s\n", clientAddr)
		conn.Close()
	}()

	clientAddr := conn.RemoteAddr().String()
	fmt.Printf("Client connected: %s\n", clientAddr)

	go s.monitorConnection(conn, clientAddr)

	for {
		reader := bufio.NewReader(conn)
		conn.Write([]byte("Choose an option: [1] Add FileHash [2] Query FileHash:\n"))
		option, _ := reader.ReadString('\n')
		option = strings.TrimSpace(option)

		switch option {
		case "1":
			conn.Write([]byte("Send FileHash:\n"))
			fileHash, _ := reader.ReadString('\n')
			fileHash = strings.TrimSpace(fileHash)

			fileInfo := helpers.FileInfo{FileHashes: []string{fileHash}}
			s.storage.AddClientInfo(clientAddr, fileInfo)
			conn.Write([]byte("FileHash added successfully.\n"))

		case "2":
			conn.Write([]byte("Send FileHash to query:\n"))
			queryHash, _ := reader.ReadString('\n')
			queryHash = strings.TrimSpace(queryHash)

			clientsWithHash := s.storage.GetClientsByHash(queryHash)
			if len(clientsWithHash) > 0 {
				conn.Write([]byte("Clients with the requested FileHash:\n"))
				for _, client := range clientsWithHash {
					conn.Write([]byte(fmt.Sprintf("IP: %s\n", client)))
				}
			} else {
				conn.Write([]byte("No clients found with the requested FileHash.\n"))
			}

		default:
			conn.Write([]byte("Invalid option. Try again.\n"))
		}
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
