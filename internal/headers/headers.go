package headers

import (
	"fmt"
	"regexp"
	"strings"
)

var regex = regexp.MustCompile(`[^a-z0-9!#$%&'*+\-.^_` + "`" + `|~]`)

type Headers map[string]string

func NewHeaders() Headers {
	return map[string]string{}
}

func (h Headers) Parse(data []byte) (int, bool, error) {
	parts := strings.Split(string(data), "\r\n")
	if len(parts) < 2 {
		return 0, false, nil
	}

	// end of headers found
	if parts[0] == "" {
		return 2, true, nil
	}

	// no whitespace between field name, colon and field value is valid
	headerParts := strings.SplitN(parts[0], ":", 2)
	if len(headerParts) != 2 {
		return 0, false, fmt.Errorf("field name or field value missing: %s", parts[0])
	}
	if strings.HasSuffix(headerParts[0], " ") {
		return 0, false, fmt.Errorf("whitespace between field name and colon detected: %s", parts[0])
	}

	key := strings.ToLower(strings.TrimSpace(headerParts[0]))
	if regex.MatchString(key) {
		return 0, false, fmt.Errorf("invalid character in field name detected: %s", key)
	}

	// duplicate field names are valid
	if _, ok := h[key]; ok {
		newValue := fmt.Sprintf("%s, %s", h[key], strings.TrimSpace(headerParts[1]))
		h[key] = newValue
	} else {
		h[key] = strings.TrimSpace(headerParts[1])
	}

	if len(parts) >= 3 && parts[1] == "" {
		// if a header and the end of headers are on the same line
		return len(parts[0]) + 4, true, nil
	}

	return len(parts[0]) + 2, false, nil
}

func (h Headers) Get(key string) (string, error) {
	if _, ok := h[strings.ToLower(key)]; !ok {
		return "", fmt.Errorf("%s header does not exist", key)
	}

	return h[strings.ToLower(key)], nil
}
