package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to the server!")

	// Create a reader for standard input
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter message: ")
		msg, _ := reader.ReadString('\n')

		// Send the message to the server
		_, err := conn.Write([]byte(msg))
		if err != nil {
			fmt.Println("Error sending message:", err)
			return
		}
	}
}
