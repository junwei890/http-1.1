package request

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
	Initialised bool
	Done        bool
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
	if r.Initialised {
		rl, n, err := parseRequestLine(string(data))
		if err != nil {
			return 0, err
		}
		if n == 0 {
			// if no bytes were parsed
			return 0, nil
		}

		r.RequestLine = *rl
		r.Done = true

		return n, nil
	}

	if r.Done {
		return 0, errors.New("reading data in a done state")
	}

	return 0, errors.New("unknown state")
}

func parseRequestLine(input string) (*RequestLine, int, error) {
	requestParts := strings.Split(input, "\r\n")
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
	}, len([]byte(input)), nil
}

func RequestParser(reader io.Reader) (*Request, error) {
	buffer := make([]byte, 8)
	read := 0
	req := &Request{
		Initialised: true,
	}

	for !req.Done {
		if len(buffer) == cap(buffer) {
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
			req.Done = true
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
