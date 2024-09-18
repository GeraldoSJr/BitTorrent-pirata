package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"
	"io"
	"os"
)

type Server struct {
	sync.Mutex
	hashMap    map[int][]string
	clientData map[string][]int
}

func NewServer() *Server {
	return &Server{
		hashMap:    make(map[int][]string),
		clientData: make(map[string][]int),
	}
}

func (s *Server) handleDownloadRequest(conn net.Conn) {
    var chunkHash int
    decoder := gob.NewDecoder(conn)

    log.Println("Waiting to decode chunk hash...")
    if err := decoder.Decode(&chunkHash); err != nil {
        log.Println("Error decoding chunk hash:", err)
        return
    }
    log.Printf("Received request for chunk with hash %d\n", chunkHash)

  
    filePath := "./dataset/rayane.txt"
    file, err := os.Open(filePath)
    if err != nil {
        log.Printf("Error opening file %s: %v", filePath, err)
        return
    }
    defer file.Close()

    chunkSize := 1024
    buffer := make([]byte, chunkSize)

  
    offset := chunkHash * chunkSize
    log.Printf("Seeking to offset %d in file %s\n", offset, filePath)

    _, err = file.Seek(int64(offset), 0)
    if err != nil {
        log.Printf("Error seeking file %s: %v", filePath, err)
        return
    }

  
    bytesRead, err := file.Read(buffer)
    if err != nil && err != io.EOF {
        log.Printf("Error reading file chunk: %v", err)
        return
    }

    if bytesRead == 0 {
        log.Printf("No more data to read from file %s\n", filePath)
        return
    }

    chunkData := buffer[:bytesRead]

    encoder := gob.NewEncoder(conn)
    log.Println("Sending chunk data to client...")
    if err := encoder.Encode(chunkData); err != nil {
        log.Println("Error encoding chunk data:", err)
        return
    }
    log.Printf("Chunk with hash %d sent to client\n", chunkHash)
}




func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		clientIP := conn.RemoteAddr().String()
		s.cleanupClientData(clientIP)
		conn.Close()
	}()

	for {
		var requestType string
		decoder := gob.NewDecoder(conn)
		if err := decoder.Decode(&requestType); err != nil {
			if err.Error() == "EOF" {
				log.Println("Client disconnected")
				return
			}
			log.Println("Error decoding request type:", err)
			return
		}

		switch requestType {
		case "store":
			s.handleStoreRequest(conn)
		case "create":
			s.handleCreateRequest(conn)
		case "delete":
			s.handleDeleteRequest(conn)
		case "query":
			s.handleQueryRequest(conn)
		case "download": 
			s.handleDownloadRequest(conn)
		default:
			log.Println("Unknown request type:", requestType)
		}
	}
}

func (s *Server) handleStoreRequest(conn net.Conn) {
	var clientHashes []int
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&clientHashes); err != nil {
		log.Println("Error decoding data:", err)
		return
	}

	s.Lock()
	clientIP := conn.RemoteAddr().String()
	for _, hash := range clientHashes {
		if _, exists := s.hashMap[hash]; !exists {
			s.hashMap[hash] = []string{}
		}
		s.hashMap[hash] = append(s.hashMap[hash], clientIP)
		s.clientData[clientIP] = append(s.clientData[clientIP], hash)
	}
	s.Unlock()

	printHashMap(s.hashMap)
}

func (s *Server) handleCreateRequest(conn net.Conn) {
	var fileHash int
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&fileHash); err != nil {
		log.Println("Error decoding file hash:", err)
		return
	}

	clientIP := conn.RemoteAddr().String()

	s.Lock()
	if _, exists := s.hashMap[fileHash]; !exists {
		s.hashMap[fileHash] = []string{}
	}
	s.hashMap[fileHash] = append(s.hashMap[fileHash], clientIP)
	s.clientData[clientIP] = append(s.clientData[clientIP], fileHash)
	s.Unlock()

	fmt.Printf("File created by %s: Hash %d\n", clientIP, fileHash)
}

func (s *Server) handleDeleteRequest(conn net.Conn) {
	var fileHash int
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&fileHash); err != nil {
		log.Println("Error decoding file hash:", err)
		return
	}

	clientIP := conn.RemoteAddr().String()

	s.Lock()
	if ips, exists := s.hashMap[fileHash]; exists {
		for i, ip := range ips {
			if ip == clientIP {
				s.hashMap[fileHash] = append(ips[:i], ips[i+1:]...)
				break
			}
		}
		if len(s.hashMap[fileHash]) == 0 {
			delete(s.hashMap, fileHash)
		}
	}

	s.clientData[clientIP] = removeFromSlice(s.clientData[clientIP], fileHash)
	s.Unlock()

	fmt.Printf("File deleted by %s: Hash %d\n", clientIP, fileHash)
}

func (s *Server) handleQueryRequest(conn net.Conn) {
	var hash int
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&hash); err != nil {
		log.Println("Error decoding hash:", err)
		return
	}

	s.Lock()
	ips := s.hashMap[hash]
	s.Unlock()

	encoder := gob.NewEncoder(conn)
	encoder.Encode(ips)
}

func (s *Server) cleanupClientData(clientIP string) {
	s.Lock()
	defer s.Unlock()

	hashes, exists := s.clientData[clientIP]
	if !exists {
		return
	}

	for _, hash := range hashes {
		ips := s.hashMap[hash]
		for i, ip := range ips {
			if ip == clientIP {
				s.hashMap[hash] = append(ips[:i], ips[i+1:]...)
				break
			}
		}

		if len(s.hashMap[hash]) == 0 {
			delete(s.hashMap, hash)
		}
	}

	delete(s.clientData, clientIP)

	log.Printf("Cleaned up data for client: %s\n", clientIP)
}

func printHashMap(hashMap map[int][]string) {
	fmt.Println("Hash Map:")
	for hash, ips := range hashMap {
		fmt.Printf("Hash: %d\n", hash)
		fmt.Println("  IPs:")
		for _, ip := range ips {
			fmt.Printf("    %s\n", ip)
		}
	}
}

func removeFromSlice(slice []int, val int) []int {
	for i, v := range slice {
		if v == val {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func main() {
	server := NewServer()
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	fmt.Println("Server is listening on port 8080...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go server.handleConnection(conn)
	}
}
