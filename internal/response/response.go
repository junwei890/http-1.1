package response

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/junwei890/http-1.1/internal/headers"
)

type WriterState string

const (
	writingStatusLine WriterState = "status line"
	writingHeaders    WriterState = "headers"
	writingBody       WriterState = "body"
)

type Writer struct {
	Response    io.Writer
	writerState WriterState
}

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

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Response:    w,
		writerState: writingStatusLine,
	}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.writerState != writingStatusLine {
		return fmt.Errorf("writing status line while in %s state", w.writerState)
	}
	defer func() { w.writerState = writingHeaders }()

	switch statusCode {
	case 200:
		if _, err := w.Response.Write([]byte("HTTP/1.1 200 OK\r\n")); err != nil {
			return err
		}
	case 400:
		if _, err := w.Response.Write([]byte("HTTP/1.1 400 Bad Request\r\n")); err != nil {
			return err
		}
	case 401:
		if _, err := w.Response.Write([]byte("HTTP/1.1 401 Unauthorized\r\n")); err != nil {
			return err
		}
	case 403:
		if _, err := w.Response.Write([]byte("HTTP/1.1 403 Forbidden\r\n")); err != nil {
			return err
		}
	case 404:
		if _, err := w.Response.Write([]byte("HTTP/1.1 404 Not Found\r\n")); err != nil {
			return err
		}
	case 500:
		if _, err := w.Response.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n")); err != nil {
			return err
		}
	default:
		// there must be a space between status code and reason phrase even if reason phrase is absent
		if _, err := w.Response.Write(fmt.Appendf([]byte{}, "HTTP/1.1 %d \r\n", statusCode)); err != nil {
			return err
		}
	}

	return nil
}

// default headers if none are set
func SetDefaultHeaders(length int) headers.Headers {
	headers := headers.NewHeaders()

	headers["Content-Length"] = strconv.Itoa(length)
	headers["Connection"] = "close"
	headers["Content-Type"] = "text/plain"

	return headers
}

// override default response headers
func OverrideDefaultHeaders(headers headers.Headers, fieldName, fieldValue string) {
	// if response body is chunked encoded, content length header should be replaced
	if fieldName == "Transfer-Encoding" && fieldValue == "chunked" {
		delete(headers, "Content-Length")
		headers[fieldName] = fieldValue
		return
	}

	for key := range headers {
		if strings.EqualFold(key, fieldName) {
			headers[key] = fieldValue
		}
	}
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.writerState != writingHeaders {
		return fmt.Errorf("writing headers while in %s state", w.writerState)
	}
	defer func() { w.writerState = writingBody }()

	for key, value := range headers {
		if _, err := w.Response.Write(fmt.Appendf([]byte{}, "%s: %s\r\n", key, value)); err != nil {
			return err
		}
	}

	// extra /r/n at the end of headers
	if _, err := w.Response.Write([]byte("\r\n")); err != nil {
		return err
	}

	return nil
}

func (w *Writer) WriteBody(body []byte) (int, error) {
	if w.writerState != writingBody {
		return 0, fmt.Errorf("writing body while in %s state", w.writerState)
	}

	n, err := w.Response.Write(body)
	if err != nil {
		return 0, err
	}

	return n, nil
}

// writes chunks as it is received
func (w *Writer) WriteChunkedBody(body []byte) (int, error) {
	if w.writerState != writingBody {
		return 0, fmt.Errorf("writing body while in %s state", w.writerState)
	}

	n := 0
	// length of chunk should be in hexadecimal
	bytesWritten, err := w.Response.Write(fmt.Appendf([]byte{}, "%X\r\n", len(body)))
	if err != nil {
		return 0, err
	}
	n += bytesWritten

	bytesWritten, err = w.Response.Write(body)
	if err != nil {
		return 0, err
	}
	n += bytesWritten

	// always end chunk with \r\n
	bytesWritten, err = w.Response.Write([]byte("\r\n"))
	if err != nil {
		return 0, err
	}
	n += bytesWritten

	return n, nil
}

// this ends the entire chunked body with a 0 length of chunk and a terminating \r\n
func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.writerState != writingBody {
		return 0, fmt.Errorf("writing body while in %s state", w.writerState)
	}

	n, err := w.Response.Write([]byte("0\r\n\r\n"))
	if err != nil {
		return 0, err
	}

	return n, err
}
