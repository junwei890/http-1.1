package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/junwei890/http-1.1/internal/request"
	"github.com/junwei890/http-1.1/internal/response"
)

type Handler func(w *response.Writer, r *request.Request)

type Server struct {
	handler  Handler
	listener net.Listener
	closed   atomic.Bool
}

// #nosec G104
func (s *Server) handle(conn net.Conn) {
	defer func() {
		log.Printf("connection with %s, closed", conn.RemoteAddr().String())
		conn.Close()
	}()

	w := response.NewWriter(conn)
	// parse incoming requests with the parser written earlier
	req, err := request.RequestParser(conn)
	if err != nil {
		// if parsing fails, respond with 400
		w.WriteStatusLine(response.StatusBadRequest)

		responseBody := []byte(err.Error())

		headers := response.SetDefaultHeaders(len(responseBody))
		w.WriteHeaders(headers)

		w.WriteBody(responseBody)

		return
	}

	s.handler(w, req)
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// distinguishes between graceful shutdown and unexpected errors
			if s.closed.Load() {
				return
			}

			log.Printf("couldn't accept connection: %v", err)
			continue
		}
		log.Printf("connection with %s, accepted", conn.RemoteAddr().String())

		// handle in a goroutine so server can accept more connections
		go s.handle(conn)
	}
}

func (s *Server) Close() error {
	s.closed.Store(true)

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return fmt.Errorf("couldn't shutdown server properly: %v", err)
		}

		return nil
	}

	return nil
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("couldn't setup a listener: %v", err)
	}

	s := &Server{
		handler:  handler,
		listener: listener,
	}
	go s.listen()

	// returns so that server can be stopped using an interrupt or termination
	return s, nil
}
