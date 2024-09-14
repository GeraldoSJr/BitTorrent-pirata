package helpers

import (
	"sync"
)

// Estrutura que contém apenas o hash do arquivo
type FileInfo struct {
	FileHash string
}

// Estrutura que contém o mapa de IPs para FileInfo
type IPStorage struct {
	Data map[string]FileInfo // Mapa de IPs para FileInfo
	mu   sync.Mutex
}

// Função para criar um novo FileInfo
func NewFileInfo(fileHash string) FileInfo {
	return FileInfo{
		FileHash: fileHash,
	}
}

// Função para adicionar ou atualizar um IP com suas informações de arquivo
func (s *IPStorage) AddClientInfo(ip string, fileInfo FileInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
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

// Função para retornar todos os IPs que possuem um determinado hash
func (s *IPStorage) GetClientsByHash(fileHash string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var clients []string
	for ip, info := range s.Data {
		if info.FileHash == fileHash {
			clients = append(clients, ip)
		}
	}
	return clients
}
