package helpers

import (
	"encoding/json"
	"sync"
)

// Estrutura que contém informações do arquivo
type FileInfo struct {
	FileHash string `json:"file_hash"`
	FileID   string `json:"file_id"`
}

// Estrutura que contém o mapa de IPs para FileInfo
type IPStorage struct {
	Data map[string]FileInfo // Modificado para "Data", para ser exportado
	mu   sync.Mutex
}

// Função para criar um novo FileInfo
func NewFileInfo(fileHash string, fileID string) FileInfo {
	return FileInfo{
		FileHash: fileHash,
		FileID:   fileID,
	}
}

// Função para adicionar ou atualizar um IP com suas informações de arquivo
func (s *IPStorage) AddClientInfo(ip string, fileInfo FileInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Atualiza ou adiciona as informações do cliente
	s.Data[ip] = fileInfo
}

// Função para remover o cliente do mapa
func (s *IPStorage) RemoveClient(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Data, ip)
}

// Função para retornar o mapa completo de IPs e informações de arquivos
func (s *IPStorage) GetAllClients() map[string]FileInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Data
}

// Função para deserializar um pacote JSON para um struct FileInfo
func DecodeFileInfo(data []byte) (FileInfo, error) {
	var fileInfo FileInfo
	err := json.Unmarshal(data, &fileInfo)
	return fileInfo, err
}
