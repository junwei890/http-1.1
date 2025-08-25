package headers

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

var regex = regexp.MustCompile(`[^a-z0-9!#$%&'*+\-.^_` + "`" + `|~]`)

type Headers map[string]string

func NewHeaders() Headers {
	return map[string]string{}
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	idx := bytes.Index(data, []byte("\r\n"))
	if idx == -1 {
		return 0, false, nil
	}
	if idx == 0 {
		// end of headers found
		return 2, true, nil
	}

	// 2 substrings because whitespace is optional
	parts := bytes.SplitN(data[:idx], []byte(":"), 2)
	key := strings.ToLower(string(parts[0]))
	if key != strings.TrimRight(key, " ") {
		// whitespace allowed only before field name
		return 0, false, fmt.Errorf("whitespace between field name and colon detected: %s", key)
	}

	key = strings.TrimSpace(key)
	value := strings.TrimSpace(string(parts[1]))

	if regex.MatchString(strings.TrimSpace(key)) {
		return 0, false, fmt.Errorf("invalid character in field name detected: %s", key)
	}

	if _, ok := h[key]; ok {
		newValue := fmt.Sprintf("%s, %s", h[key], value)
		h[key] = newValue
	} else {
		h[key] = value
	}

	return idx + 2, false, nil
}

func (h Headers) Get(key string) (string, error) {
	if _, ok := h[strings.ToLower(key)]; !ok {
		return "", fmt.Errorf("%s header does not exist", key)
	}

	return h[strings.ToLower(key)], nil
}
