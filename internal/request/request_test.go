package request

import (
	"io"
	"testing"

	"github.com/junwei890/http-1.1/internal/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

// simulates reading a variable number of bytes per chunk from a network connection
func (cr *chunkReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}

	endIndex := cr.pos + cr.numBytesPerRead
	endIndex = min(endIndex, len(cr.data))

	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n

	return n, nil
}

func TestRequestLineHeaderParse(t *testing.T) {
	// test: good get request line, no headers
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err := RequestParser(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
	assert.Equal(t, headers.Headers{}, r.Headers)

	// test: good get request line, good headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost:localhost:42069\r\nUser-Agent:curl/7.81.0\r\nAccept:*/*\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err = RequestParser(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
	assert.Equal(t, "localhost:42069", r.Headers["host"])
	assert.Equal(t, "curl/7.81.0", r.Headers["user-agent"])
	assert.Equal(t, "*/*", r.Headers["accept"])

	// test: good get request line with path, good headers with whitespace
	reader = &chunkReader{
		data:            "GET /cats HTTP/1.1\r\n Host: localhost:42069 \r\n User-Agent: curl/7.81.0 \r\n Accept: */* \r\n\r\n",
		numBytesPerRead: 4,
	}
	r, err = RequestParser(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/cats", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
	assert.Equal(t, "localhost:42069", r.Headers["host"])
	assert.Equal(t, "curl/7.81.0", r.Headers["user-agent"])
	assert.Equal(t, "*/*", r.Headers["accept"])

	// test: good post request line with path, good headers with whitespace
	reader = &chunkReader{
		data:            "POST /cats HTTP/1.1\r\n Host: localhost:42069 \r\n User-Agent: curl/7.81.0 \r\n Accept: */* \r\n\r\n",
		numBytesPerRead: 8,
	}
	r, err = RequestParser(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "POST", r.RequestLine.Method)
	assert.Equal(t, "/cats", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
	assert.Equal(t, "localhost:42069", r.Headers["host"])
	assert.Equal(t, "curl/7.81.0", r.Headers["user-agent"])
	assert.Equal(t, "*/*", r.Headers["accept"])

	// test: good get request line with path, good duplicate headers
	reader = &chunkReader{
		data:            "GET /cats HTTP/1.1\r\nHost: localhost:42069\r\nContent-Type: application/json\r\nContent-Type: text/html\r\n\r\n",
		numBytesPerRead: 8,
	}
	r, err = RequestParser(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "GET", r.RequestLine.Method)
	assert.Equal(t, "/cats", r.RequestLine.RequestTarget)
	assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
	assert.Equal(t, "localhost:42069", r.Headers["host"])
	assert.Equal(t, "application/json, text/html", r.Headers["content-type"])

	// test: invalid number of parts in request line
	reader = &chunkReader{
		data:            "/cats HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 16,
	}
	_, err = RequestParser(reader)
	require.Error(t, err)

	// test: request line out of order
	reader = &chunkReader{
		data:            "/cats GET HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 32,
	}
	_, err = RequestParser(reader)
	require.Error(t, err)

	// test: invalid http version
	reader = &chunkReader{
		data:            "GET /cats HTTP/1.2\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 64,
	}
	_, err = RequestParser(reader)
	require.Error(t, err)

	// test: whitespace between field name and colon
	reader = &chunkReader{
		data:            "GET /cats HTTP/1.1\r\nHost : localhost:42069\r\n\r\n",
		numBytesPerRead: 64,
	}
	_, err = RequestParser(reader)
	require.Error(t, err)

	// test: invalid character in field name
	reader = &chunkReader{
		data:            "GET /cats HTTP/1.1\r\nH<st: localhost:42069\r\n\r\n",
		numBytesPerRead: 1,
	}
	_, err = RequestParser(reader)
	require.Error(t, err)
}

func TestBodyParse(t *testing.T) {
	// test: valid body and content length
	reader := &chunkReader{
		data:            "GET /cats HTTP/1.1\r\nHost: localhost:42069\r\nContent-Length: 12\r\n\r\nhello world\n",
		numBytesPerRead: 4,
	}
	r, err := RequestParser(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "hello world\n", string(r.Body))
	assert.Equal(t, 12, len(r.Body))

	// test: empty body with no content length
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\n\r\n",
		numBytesPerRead: 8,
	}
	r, err = RequestParser(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))
	assert.Equal(t, 0, len(r.Body))

	// test: empty body with content length
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nContent-Length: 0\r\n\r\n",
		numBytesPerRead: 8,
	}
	r, err = RequestParser(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))
	assert.Equal(t, 0, len(r.Body))

	// test: incomplete request
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nContent-Length: 69\r\n\r\nhello world\n",
		numBytesPerRead: 16,
	}
	_, err = RequestParser(reader)
	require.Error(t, err)

	// test: no content length but body exists
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\n\r\nhello world\n",
		numBytesPerRead: 16,
	}
	r, err = RequestParser(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))
	assert.Equal(t, 0, len(r.Body))
}
