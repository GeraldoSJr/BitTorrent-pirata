package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Client struct {
	sync.Mutex
	hashMap map[int]string
}

func NewClient() *Client {
	return &Client{
		hashMap: make(map[int]string),
	}
}

func readFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file %s: %v", filePath, err)
		return nil, err
	}
	return data, nil
}

func sum(filePath string) (int, error) {
	data, err := readFile(filePath)
	if err != nil {
		return 0, err
	}

	_sum := 0
	for _, b := range data {
		_sum += int(b)
	}

	return _sum, nil
}

func sumWrapper(filePath string, sumChan chan struct {
	int
	string
}) {
	fileSum, err := sum(filePath)

	if err != nil {
		fileSum = 0
	}

	sumChan <- struct {
		int
		string
	}{fileSum, filePath}
}

func listarArquivos(diretorio string) []string {
	arquivos, err := os.ReadDir(diretorio)
	if err != nil {
		log.Fatal(err)
	}

	var result []string
	for _, arquivo := range arquivos {
		if !arquivo.IsDir() {
			result = append(result, arquivo.Name())
		}
	}
	return result
}

func generateFilesHashMap(diretorio string) map[string][]int {
	if _, err := os.Stat(diretorio); os.IsNotExist(err) {
		log.Fatalf("O diretório %s não existe", diretorio)
	}
	files := listarArquivos(diretorio)
	size := len(files)
	sumChannel := make(chan struct {
		int
		string
	}, size)

	for _, path := range files {
		go sumWrapper(filepath.Join(diretorio, path), sumChannel)
	}

	hashs := make(map[string][]int)
	for i := 0; i < size; i++ {
		result := <-sumChannel
		hashs[result.string] = append(hashs[result.string], result.int)
	}

	return hashs
}

func storeHashes(conn net.Conn, hashes map[string][]int) {
	encoder := gob.NewEncoder(conn)

	if err := encoder.Encode("store"); err != nil {
		log.Println("Error encoding request type:", err)
		return
	}

	var hashList []int
	for _, v := range hashes {
		hashList = append(hashList, v...)
	}

	if err := encoder.Encode(hashList); err != nil {
		log.Println("Error encoding hashes:", err)
		return
	}

	fmt.Println("Initial hashes stored.")
}

func updateServer(conn net.Conn, action string, filePath string, client *Client) {
	encoder := gob.NewEncoder(conn)

	if err := encoder.Encode(action); err != nil {
		log.Println("Error encoding action:", err)
		return
	}

	fileHash, err := sum(filePath)
	if err != nil {
		log.Printf("Error calculating hash for file %s: %v", filePath, err)
		fileHash = 0
	}

	if err := encoder.Encode(fileHash); err != nil {
		log.Println("Error encoding file hash:", err)
		return
	}

	client.Lock()
	client.hashMap[fileHash] = filePath
	client.Unlock()

	fmt.Printf("Server updated: %s - %s\n", action, filePath)
}

func monitorDirectory(conn net.Conn, directory string, server *Client) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(directory)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				fmt.Println("File created:", event.Name)
				updateServer(conn, "create", event.Name, server)
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				fmt.Println("File deleted:", event.Name)
				updateServer(conn, "delete", event.Name, server)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("Error:", err)
		}
	}
}

func querySingleHash(conn net.Conn, hash int) ([]string, error) {
	encoder := gob.NewEncoder(conn)

	if err := encoder.Encode("query"); err != nil {
		log.Println("Error encoding request type:", err)
		return nil, err
	}

	if err := encoder.Encode(hash); err != nil {
		log.Println("Error encoding hash:", err)
		return nil, err
	}

	decoder := gob.NewDecoder(conn)
	var ips []string
	if err := decoder.Decode(&ips); err == nil {
		fmt.Println("IPs for hash", hash, ":", ips)
	} else {
		log.Println("Error decoding result:", err)
	}

	return ips, nil
}

func (s *Client) handleStoreRequest(conn net.Conn) {
	storageDir := "./storage"
	err := os.MkdirAll(storageDir, os.ModePerm)
	if err != nil {
		log.Println("Error creating storage directory:", err)
		return
	}

	var filename string
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&filename); err != nil {
		log.Println("Error decoding filename:", err)
		return
	}

	var chunkData []byte
	if err := decoder.Decode(&chunkData); err != nil {
		log.Println("Error decoding chunk data:", err)
		return
	}

	filePath := filepath.Join(storageDir, filename)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Error opening/creating file:", err)
		return
	}
	defer file.Close()

	if _, err := file.Write(chunkData); err != nil {
		log.Println("Error writing chunk to file:", err)
		return
	}

	log.Printf("Chunk stored successfully in %s\n", filePath)
}

func (s *Client) handleDownloadRequest(conn net.Conn) {
	// Decodificar o hash recebido
	var fileHash int
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&fileHash); err != nil {
		log.Println("Error decoding file hash:", err)
		return
	}

	// Construir o caminho completo do arquivo usando o hash
	filePath := s.hashMap[fileHash]
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Ler os dados do chunk
	chunkData, err := io.ReadAll(file)
	if err != nil {
		log.Println("Error reading chunk data:", err)
		return
	}

	// Enviar os dados do chunk de volta ao cliente
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(chunkData); err != nil {
		log.Println("Error encoding chunk data:", err)
		return
	}

	log.Printf("Chunk with hash %d sent successfully\n", fileHash)
}

func (s *Client) handleConnection(conn net.Conn) {
	defer conn.Close()

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
		case "download":
			s.handleDownloadRequest(conn)
		default:
			log.Println("Unknown request type:", requestType)
		}
	}
}

func startClientServer(server *Client) {
	ln, err := net.Listen("tcp", ":9090")
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

func donwload(hash int, ip string, outputPath string) error {
	// Conectar ao servidor
	conn, err := net.Dial("tcp", ip+":9090")
	if err != nil {
		return fmt.Errorf("error connecting to server: %v", err)
	}
	defer conn.Close()

	// Criar um encoder para enviar a solicitação de download
	encoder := gob.NewEncoder(conn)

	// Enviar o tipo de solicitação
	requestType := "download"
	if err := encoder.Encode(&requestType); err != nil {
		return fmt.Errorf("error sending request type: %v", err)
	}

	// Enviar o hash do arquivo
	if err := encoder.Encode(&hash); err != nil {
		return fmt.Errorf("error sending file hash: %v", err)
	}

	// Criar um decoder para receber o chunk
	decoder := gob.NewDecoder(conn)
	var chunkData []byte
	if err := decoder.Decode(&chunkData); err != nil {
		return fmt.Errorf("error receiving chunk data: %v", err)
	}

	// Salvar o chunk em um arquivo
	if err := os.WriteFile(outputPath, chunkData, 0644); err != nil {
		return fmt.Errorf("error saving chunk to file: %v", err)
	}

	fmt.Printf("Chunk downloaded and saved to %s\n", outputPath)
	return nil
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter server IP: ")
	serverIp, _ := reader.ReadString('\n')
	serverIp = strings.TrimSpace(serverIp)

	conn, err := net.Dial("tcp", serverIp+":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	directory := "./dataset/"
	initialHashes := generateFilesHashMap(directory)
	storeHashes(conn, initialHashes)

	server := NewClient()
	go monitorDirectory(conn, directory, server)
	go startClientServer(server)

	for {
		fmt.Println("\nChoose an option:")
		fmt.Println("1. Query hash")
		fmt.Println("2. Exit")
		fmt.Println("3. Download file")
		fmt.Print("Enter choice (1, 2 or 3): ")

		choiceStr, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Error reading input:", err)
		}
		choiceStr = strings.TrimSpace(choiceStr)
		choice, err := strconv.Atoi(choiceStr)
		if err != nil {
			log.Fatal("Invalid choice:", err)
		}

		switch choice {
		case 1:

			fmt.Print("Enter hash to query: ")
			hashInput, _ := reader.ReadString('\n')
			hashInput = strings.TrimSpace(hashInput)
			hash, err := strconv.Atoi(hashInput)
			if err != nil {
				log.Fatal("Invalid hash value:", err)
			}
			querySingleHash(conn, hash)

		case 2:

			fmt.Println("Exiting...")
			return

		case 3:

			fmt.Print("Enter hash to query: ")
			hashInput, _ := reader.ReadString('\n')
			hashInput = strings.TrimSpace(hashInput)
			hash, err := strconv.Atoi(hashInput)

			if err != nil {
				log.Fatal("Invalid hash value:", err)
			}

			fmt.Print("Enter file path to output: ")
			filePath, _ := reader.ReadString('\n')
			filePath = strings.TrimSpace(filePath)

			ips, err := querySingleHash(conn, hash)

			if err != nil {
				log.Fatal("Error while searching for IPs for the provided hash", err)
				continue
			}

			if len(ips) == 0 {
				fmt.Println("No IPs found for the provided hash.")
				continue
			}

			go donwload(hash, ips[0], filePath)

			fmt.Printf("File successfully downloaded and saved to %s\n", filePath)

		default:
			fmt.Println("Invalid choice. Please enter 1, 2 or 3.")
		}
	}
}
