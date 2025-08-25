package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/junwei890/http-1.1/internal/request"
	"github.com/junwei890/http-1.1/internal/response"
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
func handler(w *response.Writer, r *request.Request) {
	switch r.RequestLine.RequestTarget {
	case "/":
		w.WriteStatusLine(response.StatusOK)
		responseBody := []byte(`<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Hello</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`)

		headers := response.SetDefaultHeaders(len(responseBody))
		response.OverrideDefaultHeaders(headers, "Content-Type", "text/html")
		w.WriteHeaders(headers)

		w.WriteBody(responseBody)
	}
}
