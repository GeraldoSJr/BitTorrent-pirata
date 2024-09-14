package helpers

type FileInfo struct {
	FileHashes []string
}

type Request struct {
	Operation string
	ClientIP  string
	FileInfo  *FileInfo
	Query     string
	Response  chan []string
}

type IPStorage struct {
	requests chan Request
}

func NewIPStorage() *IPStorage {
	storage := &IPStorage{
		requests: make(chan Request, 10),
	}
	go storage.run()
	return storage
}

func (s *IPStorage) run() {
	data := make(map[string]FileInfo)

	for req := range s.requests {
		switch req.Operation {
		case "add":
			if existingInfo, exists := data[req.ClientIP]; exists {
				existingInfo.FileHashes = append(existingInfo.FileHashes, req.FileInfo.FileHashes...)
				data[req.ClientIP] = existingInfo
			} else {
				data[req.ClientIP] = *req.FileInfo
			}
		case "get":
			var clients []string
			for ip, info := range data {
				for _, hash := range info.FileHashes {
					if hash == req.Query {
						clients = append(clients, ip)
						break
					}
				}
			}
			req.Response <- clients
		case "remove":
			delete(data, req.ClientIP)
		}
	}
}

func (s *IPStorage) AddClientInfo(ip string, fileInfo FileInfo) {
	s.requests <- Request{
		Operation: "add",
		ClientIP:  ip,
		FileInfo:  &fileInfo,
	}
}

func (s *IPStorage) GetClientsByHash(fileHash string) []string {
	response := make(chan []string, 10)
	s.requests <- Request{
		Operation: "get",
		Query:     fileHash,
		Response:  response,
	}
	return <-response
}

func (s *IPStorage) RemoveClient(ip string) {
	s.requests <- Request{
		Operation: "remove",
		ClientIP:  ip,
	}
}
