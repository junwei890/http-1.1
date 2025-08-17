package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Connection accepted")

		lines := readRequest(conn) // conn implements read and write
		for line := range lines {
			fmt.Println(line)
		}

		conn.Close()
	}
}

func readRequest(r io.ReadCloser) <-chan string {
	channel := make(chan string)

	go func() {
		defer close(channel)

		line := ""
		for {
			buffer := make([]byte, 8)
			if _, err := r.Read(buffer); err != nil && err != io.EOF {
				log.Fatal(err)
			} else if err == io.EOF { // eof is for when no bytes are read
				break
			}

			parts := strings.Split(string(buffer), "\n")
			for i, part := range parts {
				line += part
				if i != (len(parts) - 1) {
					channel <- line
					line = ""
				}
			}
		}
	}()

	return channel
}
