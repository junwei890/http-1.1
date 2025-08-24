package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"

	"github.com/junwei890/http-1.1/internal/request"
	"github.com/junwei890/http-1.1/internal/response"
)

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

type Handler func(w io.Writer, r *request.Request) *HandlerError

type Server struct {
	handler  Handler
	listener net.Listener
	closed   atomic.Bool
}

// helper function to send responses in the event errors
// #nosec G104
func (h *HandlerError) errorResponseWriter(w io.Writer) {
	response.WriteStatusLine(w, h.StatusCode)

	defaultHeaders := response.SetDefaultHeaders(len([]byte(h.Message)))
	response.WriteHeaders(w, defaultHeaders)

	w.Write([]byte(h.Message))
}

// #nosec G104
func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	// parse incoming requests with the parser written earlier
	req, err := request.RequestParser(conn)
	if err != nil {
		errRes := &HandlerError{
			StatusCode: response.StatusBadRequest,
			Message:    err.Error(),
		}

		errRes.errorResponseWriter(conn)
		return
	}

	// writes response from handler into a buffer
	buffer := bytes.NewBuffer([]byte{})
	if handlerErr := s.handler(buffer, req); handlerErr != nil {
		handlerErr.errorResponseWriter(conn)
		return
	}

	response.WriteStatusLine(conn, response.StatusOK)

	responseBody := buffer.Bytes()
	defaultHeaders := response.SetDefaultHeaders(len(responseBody))
	response.WriteHeaders(conn, defaultHeaders)

	// use the buffer written into earlier to write response body
	conn.Write(responseBody)
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

	return errors.New("trying to shutdown a server that isn't running")
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
