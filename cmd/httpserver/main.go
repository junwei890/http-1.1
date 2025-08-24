package main

import (
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/junwei890/http-1.1/internal/request"
	"github.com/junwei890/http-1.1/internal/server"
)

const port = 42069

func main() {
	server, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("couldn't start server: %v", err)
	}
	defer server.Close()

	log.Printf("server started on port :%d\n", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("server shutdown")
}

// #nosec G104
func handler(w io.Writer, r *request.Request) *server.HandlerError {
	w.Write([]byte("All good\n"))
	return nil
}
