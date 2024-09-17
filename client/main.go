package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
)

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

func updateServer(conn net.Conn, action string, filePath string) {
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

	fmt.Printf("Server updated: %s - %s\n", action, filePath)
}

func monitorDirectory(conn net.Conn, directory string) {
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
				updateServer(conn, "create", event.Name)
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				fmt.Println("File deleted:", event.Name)
				updateServer(conn, "delete", event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("Error:", err)
		}
	}
}

func queryHash(conn net.Conn, hash int) {
	encoder := gob.NewEncoder(conn)

	if err := encoder.Encode("query"); err != nil {
		log.Println("Error encoding request type:", err)
		return
	}

	if err := encoder.Encode(hash); err != nil {
		log.Println("Error encoding hash:", err)
		return
	}

	decoder := gob.NewDecoder(conn)
	var ips []string
	if err := decoder.Decode(&ips); err == nil {
		fmt.Println("IPs for hash", hash, ":", ips)
	} else {
		log.Println("Error decoding result:", err)
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	directory := "./dataset/"
	initialHashes := generateFilesHashMap(directory)
	storeHashes(conn, initialHashes)

	go monitorDirectory(conn, directory)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\nChoose an option:")
		fmt.Println("1. Query hash")
		fmt.Println("2. Exit")
		fmt.Print("Enter choice (1 or 2): ")

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
			queryHash(conn, hash)

		case 2:
			fmt.Println("Exiting...")
			return

		default:
			fmt.Println("Invalid choice. Please enter 1 or 2.")
		}
	}
}
