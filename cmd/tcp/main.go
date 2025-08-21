package main

import (
	"fmt"
	"log"
	"net"

	"github.com/junwei890/http-1.1/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069") // #nosec G102
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("Listening for requests")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Connection accepted")

		req, err := request.RequestParser(conn)
		if err != nil {
			log.Println(err)
		}

		fmt.Println("Request line:")
		fmt.Printf("- Method: %s\n", req.RequestLine.Method)
		fmt.Printf("- Target: %s\n", req.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", req.RequestLine.HttpVersion)

		if err := conn.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
