package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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

func generateFilesHashMap() map[string][]int {
	diretorio := "./dataset/"

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
		go sumWrapper((diretorio + path), sumChannel)
	}

	hashs := make(map[string][]int)
	for i := 0; i < size; i++ {
		result := <-sumChannel
		hashs[result.string] = append(hashs[result.string], result.int)
	}

	return hashs
}

func storeHashes(conn net.Conn, hashes []int) {
	encoder := gob.NewEncoder(conn)

	if err := encoder.Encode("store"); err != nil {
		log.Println("Error encoding request type:", err)
		return
	}

	if err := encoder.Encode(hashes); err != nil {
		log.Println("Error encoding hashes:", err)
		return
	}

	fmt.Println("Hashes stored.")
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

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\nChoose an option:")
		fmt.Println("1. Store hashes")
		fmt.Println("2. Query hash")
		fmt.Println("3. Exit")
		fmt.Print("Enter choice (1, 2, or 3): ")

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
			fileHashs := generateFilesHashMap()
			fmt.Print(fileHashs)
			var hashes []int
			for _, ints := range fileHashs {
				hashes = append(hashes, ints...)
			}
			fmt.Print(hashes)
			storeHashes(conn, hashes)

		case 2:
			fmt.Print("Enter hash to query: ")
			hashInput, _ := reader.ReadString('\n')
			hashInput = strings.TrimSpace(hashInput)
			hash, err := strconv.Atoi(hashInput)
			if err != nil {
				log.Fatal("Invalid hash value:", err)
			}
			queryHash(conn, hash)

		case 3:
			fmt.Println("Exiting...")
			return

		default:
			fmt.Println("Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}
