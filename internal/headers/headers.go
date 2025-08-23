package headers

import (
	"fmt"
	"regexp"
	"strings"
)

// regex to match characters not in set
var regex = regexp.MustCompile(`[^a-z0-9!#$%&'*+\-.^_` + "`" + `|~]`)

type Headers map[string]string

func NewHeaders() Headers {
	return map[string]string{}
}

// called continuously until done is true
func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	parts := strings.Split(string(data), "\r\n")
	if len(parts) < 2 {
		return 0, false, nil
	}

	// end of headers found
	if parts[0] == "" {
		return 2, true, nil
	}

	// host:localhost:42069 is valid
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

	// +2 to account for \r\n
	return len(parts[0]) + 2, false, nil
}
