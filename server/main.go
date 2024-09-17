package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"
)

type Server struct {
    sync.Mutex
    hashMap map[int][]string
}

func NewServer() *Server {
    return &Server{
        hashMap: make(map[int][]string),
    }
}

func (s *Server) handleConnection(conn net.Conn) {
    defer conn.Close()

    var requestType string
    decoder := gob.NewDecoder(conn)
    if err := decoder.Decode(&requestType); err != nil {
        log.Println("Error decoding request type:", err)
        return
    }

    if requestType == "store" {
        s.handleStoreRequest(conn)
    } else if requestType == "query" {
        s.handleQueryRequest(conn)
    } else {
        log.Println("Unknown request type:", requestType)
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
    }
    s.Unlock()

    printHashMap(s.hashMap)  // Use the custom print function
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
