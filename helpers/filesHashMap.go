package helpers

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
    arquivos, err := ioutil.ReadDir(diretorio)
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
    diretorio := "../dataset/" 

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
		go sumWrapper((diretorio+path), sumChannel)
	}

	hashs := make(map[string][]int)
	for i := 0; i < size; i++ {
		result := <-sumChannel
		hashs[result.string] = append(hashs[result.string], result.int)
	}

	return hashs

}
