package helpers

type FileInfo struct {
	FileHash string
}

type Request struct {
	Operation string         // "add" ou "get"
	ClientIP  string         // Endereço IP do cliente
	FileInfo  *FileInfo      // Dados do FileInfo para adicionar
	Query     string         // Hash a ser consultado
	Response  chan []string  // Canal para enviar a resposta
}

type IPStorage struct {
	requests chan Request // Canal para enviar pedidos de acesso ao mapa
}

// Função para inicializar a estrutura IPStorage com o canal
func NewIPStorage() *IPStorage {
	storage := &IPStorage{
		requests: make(chan Request),
	}
	go storage.run()
	return storage
}

// Função que a goroutine principal executa para processar requests
func (s *IPStorage) run() {
	data := make(map[string]FileInfo)

	for req := range s.requests {
		switch req.Operation {
		case "add":
			data[req.ClientIP] = *req.FileInfo
		case "get":
			var clients []string
			for ip, info := range data {
				if info.FileHash == req.Query {
					clients = append(clients, ip)
				}
			}
			req.Response <- clients // Enviar a resposta pelo canal
		}
	}
}

// Adiciona um cliente e seu hash no mapa
func (s *IPStorage) AddClientInfo(ip string, fileInfo FileInfo) {
	s.requests <- Request{
		Operation: "add",
		ClientIP:  ip,
		FileInfo:  &fileInfo,
	}
}

// Obtém os clientes associados a um hash
func (s *IPStorage) GetClientsByHash(fileHash string) []string {
	response := make(chan []string)
	s.requests <- Request{
		Operation: "get",
		Query:     fileHash,
		Response:  response,
	}
	return <-response
}
