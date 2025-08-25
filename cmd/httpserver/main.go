package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/junwei890/http-1.1/internal/headers"
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
	if r.RequestLine.RequestTarget == "/" {
		w.WriteStatusLine(response.StatusOK)

		responseBody := []byte(
			`<html>
  <head>
    <title>Server</title>
  </head>
  <body>
    <h1>Hello World!</h1>
  </body>
</html>`)

		headers := response.SetDefaultHeaders(len(responseBody))
		response.OverrideDefaultHeaders(headers, "Content-Type", "text/html")
		w.WriteHeaders(headers)

		w.WriteBody(responseBody)
	} else if route, ok := strings.CutPrefix(r.RequestLine.RequestTarget, "/httpbin/"); ok {
		// proxy for https://httpbin.org/ and a testing endpoint for chunked encoding and trailers
		url := fmt.Sprintf("https://httpbin.org/%s", route)

		client := &http.Client{}
		res, err := client.Get(url)
		if err != nil {
			errorResponseHandler(w, r, err)
			return
		}
		defer res.Body.Close()

		w.WriteStatusLine(response.StatusOK)

		h := response.SetDefaultHeaders(0)
		// trailers must be declared in the headers
		response.OverrideDefaultHeaders(h, "Trailers", "X-Content-SHA256, X-Content-Length")
		response.OverrideDefaultHeaders(h, "Transfer-Encoding", "chunked")
		w.WriteHeaders(h)

		// keep reading from response till it ends, in real time
		responseBody := []byte{}
		buffer := make([]byte, 64)
		for {
			n, err := res.Body.Read(buffer)
			if err != nil && err != io.EOF {
				log.Printf("couldn't read response from %s: %v", url, err)
				break
			}
			if err == io.EOF {
				break
			}
			if n > 0 {
				responseBody = slices.Concat(responseBody, buffer[:n])
				if _, err := w.WriteChunkedBody(buffer[:n]); err != nil {
					log.Printf("couldn't write chunked body from %s: %v", url, err)
					break
				}
			}
		}

		if _, err := w.WriteChunkedBodyDone(); err != nil {
			log.Printf("couldn't terminate chunked body for %s: %v", url, err)
		}

		trailers := headers.NewHeaders()

		hash := sha256.Sum256(responseBody)
		contentLength := len(responseBody)
		response.OverrideDefaultHeaders(trailers, "X-Content-SHA256", fmt.Sprintf("%x", hash))
		response.OverrideDefaultHeaders(trailers, "X-Content-Length", strconv.Itoa(contentLength))

		if err := w.WriteTrailers(trailers); err != nil {
			log.Printf("couldn't write trailers for %s: %v", url, err)
		}
	}
}

// #nosec G104
func errorResponseHandler(w *response.Writer, _ *request.Request, err error) {
	w.WriteStatusLine(response.StatusInternalServerError)

	responseBody := []byte(err.Error())

	headers := response.SetDefaultHeaders(len(responseBody))
	w.WriteHeaders(headers)

	w.WriteBody(responseBody)
}
