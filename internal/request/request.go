package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/junwei890/http-1.1/internal/headers"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type parserState string

const (
	parsingRequestLine parserState = "request line"
	parsingHeaders     parserState = "headers"
	parsingBody        parserState = "body"
	parsingDone        parserState = "done"
)

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
	bodyLength  int
	state       parserState
}

var validMethods map[string]struct{} = map[string]struct{}{
	"GET":     {},
	"POST":    {},
	"PUT":     {},
	"PATCH":   {},
	"DELETE":  {},
	"HEAD":    {},
	"OPTIONS": {},
	"CONNECT": {},
	"TRACE":   {},
}

func RequestParser(reader io.Reader) (*Request, error) {
	buffer := make([]byte, 8)
	read := 0
	req := &Request{
		state:   parsingRequestLine,
		Headers: headers.NewHeaders(),
	}

	for req.state != parsingDone {
		// if there is the case of multiple reads without parsing and the buffer is full
		if read >= len(buffer) {
			newBuffer := make([]byte, len(buffer)*2)
			copy(newBuffer, buffer)
			buffer = newBuffer
		}

		// read into section after unparsed bytes
		bytesRead, err := reader.Read(buffer[read:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				if req.state != parsingDone {
					// for when content length specified is larger than length of body received
					return nil, errors.New("incomplete request")
				}
				break
			}
			return nil, err
		}
		read += bytesRead

		// try to parse the buffer
		bytesParsed, err := req.parse(buffer[:read])
		if err != nil {
			return nil, err
		}
		// if anything is parsed, parsed bytes are cleaned
		copy(buffer, buffer[bytesParsed:])
		read -= bytesParsed
	}

	return req, nil
}

// only parses when it receives the entire request line
func parseRequestLine(data []byte) (*RequestLine, int, error) {
	i := bytes.Index(data, []byte("\r\n"))
	if i == -1 {
		return nil, 0, nil
	}

	requestLine := string(data[:i])

	// all 3 parts of a request line are required
	requestLineParts := strings.Split(requestLine, " ")
	if len(requestLineParts) != 3 {
		return nil, 0, fmt.Errorf("request line requires 3 parts, only have %d", len(requestLineParts))
	}

	// formatting checks
	if _, ok := validMethods[requestLineParts[0]]; !ok {
		return nil, 0, fmt.Errorf("%s method not supported", requestLineParts[0])
	}

	if !strings.HasPrefix(requestLineParts[1], "/") {
		return nil, 0, fmt.Errorf("%s is an invalid route", requestLineParts[1])
	}

	if requestLineParts[2] != "HTTP/1.1" {
		return nil, 0, fmt.Errorf("%s is an unsupported protocol or version", requestLineParts[2])
	}

	return &RequestLine{
		HttpVersion:   strings.Split(requestLineParts[2], "/")[1],
		RequestTarget: requestLineParts[1],
		Method:        requestLineParts[0],
	}, i + 2, nil
}

func (r *Request) parse(data []byte) (int, error) {
	bytesParsed := 0
	for r.state != parsingDone {
		n, err := r.parseHelper(data[bytesParsed:])
		if err != nil {
			return 0, err
		}

		bytesParsed += n
		if n == 0 {
			break
		}
	}

	return bytesParsed, nil
}

func (r *Request) parseHelper(data []byte) (int, error) {
	switch r.state {
	case parsingRequestLine:
		requestLine, n, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, nil
		}

		r.RequestLine = *requestLine
		r.state = parsingHeaders

		return n, nil
	case parsingHeaders:
		// unlike request line, headers are parsed one by one
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.state = parsingBody
		}

		return n, nil
	case parsingBody:
		// keeps adding to body until we reach eof or when entire body has been received
		lengthString, err := r.Headers.Get("Content-Length")
		if err != nil {
			r.state = parsingDone
			return len(data), nil
		}

		lengthInt, err := strconv.Atoi(lengthString)
		if err != nil {
			return 0, fmt.Errorf("%s not a valid content length", lengthString)
		}

		r.Body = slices.Concat(r.Body, data)
		r.bodyLength += len(data)
		if r.bodyLength > lengthInt {
			return 0, fmt.Errorf("length of body: %d, is more than content length specified: %d", len(r.Body), lengthInt)
		}
		if r.bodyLength == lengthInt {
			r.state = parsingDone
		}

		return len(data), nil
	case parsingDone:
		return 0, fmt.Errorf("parsing in a done state")
	default:
		return 0, fmt.Errorf("unknown state")
	}
}

