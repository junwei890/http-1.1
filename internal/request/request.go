package request

import (
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
	stateRequestLine parserState = "request line"
	stateHeaders     parserState = "headers"
	stateBody        parserState = "body"
	stateDone        parserState = "done"
)

type Request struct {
	RequestLine        RequestLine
	Headers            headers.Headers
	Body               []byte
	currentParserState parserState
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
	switch r.currentParserState {
	case stateRequestLine:
		rl, n, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, nil
		}

		r.currentParserState = stateHeaders
		r.RequestLine = *rl

		return n, nil
	case stateHeaders:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			// if content length is not specified or content length is 0, parsing is done
			lengthString, err := r.Headers.Get("content-length")
			if err != nil {
				r.currentParserState = stateDone
				return n, nil
			}

			lengthInt, err := strconv.Atoi(lengthString)
			if err != nil {
				return n, fmt.Errorf("%s not a valid content length", lengthString)
			}
			if lengthInt == 0 {
				r.currentParserState = stateDone
				return n, nil
			}

			r.currentParserState = stateBody
		}

		return n, nil
	case stateBody:
		// already checked that content length is present and valid in previous state
		lengthString, _ := r.Headers.Get("content-length")
		lengthInt, _ := strconv.Atoi(lengthString)

		r.Body = slices.Concat(r.Body, data)

		if len(r.Body) > lengthInt {
			return 0, fmt.Errorf("length of body: %d, is more than content length specified: %d", len(r.Body), lengthInt)
		} else if len(r.Body) == lengthInt {
			r.currentParserState = stateDone
		}

		return len(data), nil
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

	// all 3 parts of a request line are required
	requestLineParts := strings.Split(requestParts[0], " ")
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
	}, len([]byte(requestParts[0])) + 2, nil
}

func RequestParser(reader io.Reader) (*Request, error) {
	buffer := make([]byte, 1024)
	read := 0
	req := &Request{
		currentParserState: stateRequestLine,
		Headers:            headers.NewHeaders(),
	}

	for req.currentParserState != stateDone {
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
			if req.currentParserState != stateDone {
				return nil, errors.New("incomplete request")
			}
			break
		}
		read += bytesRead

		bytesParsed, err := req.parse(buffer[:read])
		if err != nil {
			return nil, err
		}

		// removes parsed bytes from the buffer
		copy(buffer, buffer[bytesParsed:read])
		read -= bytesParsed
	}

	return req, nil
}
