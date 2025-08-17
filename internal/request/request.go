package request

import (
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

func parseRequestLine(requestLine string) (*RequestLine, error) {
	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		return &RequestLine{}, fmt.Errorf("request line requires 3 parts, only have %d", len(parts))
	}

	if _, ok := validMethods[parts[0]]; !ok {
		return &RequestLine{}, fmt.Errorf("%s method not supported", parts[0])
	}

	if !strings.HasPrefix(parts[1], "/") {
		return &RequestLine{}, fmt.Errorf("%s is an invalid route", parts[1])
	}

	if parts[2] != "HTTP/1.1" {
		return &RequestLine{}, fmt.Errorf("%s is an unsupported protocol or version", parts[2])
	}

	return &RequestLine{
		HttpVersion:   strings.Split(parts[2], "/")[1],
		RequestTarget: parts[1],
		Method:        parts[0],
	}, nil
}

func RequestParser(reader io.Reader) (*Request, error) {
	request, err := io.ReadAll(reader)
	if err != nil {
		return &Request{}, err
	}

	requestParts := strings.Split(string(request), "\r\n")
	if len(requestParts) < 3 {
		return &Request{}, fmt.Errorf("request requires minimum 3 parts, only have %d", len(requestParts))
	}

	requestLine, err := parseRequestLine(requestParts[0])
	if err != nil {
		return &Request{}, err
	}

	return &Request{
		RequestLine: *requestLine,
	}, nil
}
