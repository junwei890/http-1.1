package response

import (
	"fmt"
	"io"
	"strconv"

	"github.com/junwei890/http-1.1/internal/headers"
)

type StatusCode int

// only handling status codes I use most often
const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusUnauthorized        StatusCode = 401
	StatusForbidden           StatusCode = 403
	StatusNotFound            StatusCode = 404
	StatusInternalServerError StatusCode = 500
)

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	switch statusCode {
	case 200:
		if _, err := w.Write([]byte("HTTP/1.1 200 OK\r\n")); err != nil {
			return err
		}
	case 400:
		if _, err := w.Write([]byte("HTTP/1.1 400 Bad Request\r\n")); err != nil {
			return err
		}
	case 401:
		if _, err := w.Write([]byte("HTTP/1.1 401 Unauthorized\r\n")); err != nil {
			return err
		}
	case 403:
		if _, err := w.Write([]byte("HTTP/1.1 403 Forbidden\r\n")); err != nil {
			return err
		}
	case 404:
		if _, err := w.Write([]byte("HTTP/1.1 404 Not Found\r\n")); err != nil {
			return err
		}
	case 500:
		if _, err := w.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n")); err != nil {
			return err
		}
	default:
		// there must be a space between status code and reason phrase even if reason phrase is absent
		if _, err := w.Write(fmt.Appendf([]byte{}, "HTTP/1.1 %d \r\n", statusCode)); err != nil {
			return err
		}
	}

	return nil
}

func SetDefaultHeaders(length int) headers.Headers {
	headers := headers.NewHeaders()

	headers["Content-Length"] = strconv.Itoa(length)
	headers["Connection"] = "close"
	headers["Content-Type"] = "text/plain"

	return headers
}

func WriteHeaders(w io.Writer, headers headers.Headers) error {
	for key, value := range headers {
		if _, err := w.Write(fmt.Appendf([]byte{}, "%s: %s\r\n", key, value)); err != nil {
			return err
		}
	}

	// extra \r\n after end of headers
	if _, err := w.Write([]byte("\r\n")); err != nil {
		return err
	}

	return nil
}
