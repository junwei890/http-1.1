package request

import (
	"errors"
	"fmt"
	"io"
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
	stateRequestLine parserState = "request line"
	stateHeaders     parserState = "headers"
	stateDone        parserState = "done"
)

type Request struct {
	RequestLine        RequestLine
	Headers            headers.Headers
	CurrentParserState parserState
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

func (r *Request) parse(data []byte) (int, error) {
	switch r.CurrentParserState {
	case stateRequestLine:
		rl, n, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, nil
		}

		r.CurrentParserState = stateHeaders
		r.RequestLine = *rl

		return n, nil
	case stateHeaders:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.CurrentParserState = stateDone
		}

		return n, nil
	case stateDone:
		return 0, errors.New("parsing in a done state")
	default:
		return 0, errors.New("unknown state")
	}
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	requestParts := strings.Split(string(data), "\r\n")
	if len(requestParts) < 2 {
		return nil, 0, nil
	}

	requestLineParts := strings.Split(requestParts[0], " ")
	if len(requestLineParts) != 3 {
		return nil, 0, fmt.Errorf("request line requires 3 parts, only have %d", len(requestLineParts))
	}

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
	}, len([]byte(requestParts[0])) + 2, nil
}

func RequestParser(reader io.Reader) (*Request, error) {
	buffer := make([]byte, 8)
	read := 0
	req := &Request{
		CurrentParserState: stateRequestLine,
		Headers:            headers.NewHeaders(),
	}

	for req.CurrentParserState != stateDone {
		// on the previous loop, if buffer was reallocated to meet the length of unparsed bytes then size of buffer will be doubled
		if read == cap(buffer) {
			newBuffer := make([]byte, 2*len(buffer))
			copy(newBuffer, buffer)
			buffer = newBuffer
		}

		// reads sections of reader into sections of buffer as it grows
		bytesRead, err := reader.Read(buffer[read:])
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF {
			req.CurrentParserState = stateDone
			break
		}
		read += bytesRead

		bytesParsed, err := req.parse(buffer[:read])
		if err != nil {
			return nil, err
		}

		// if bytesParsed > 0, this will always turn buffer into an empty slice
		copy(buffer, buffer[bytesParsed:read])
		read -= bytesParsed
	}

	return req, nil
}
