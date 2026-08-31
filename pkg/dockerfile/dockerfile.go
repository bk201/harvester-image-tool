// Package dockerfile extracts ENV values from Dockerfile content with a
// simple line scanner, not a full parser.
package dockerfile

import (
	"bufio"
	"bytes"
	"strings"
)

// EnvValue returns the value assigned to key by the last "ENV key=value" or
// "ENV key value" instruction in content, matching Docker's last-assignment-
// wins semantics. Surrounding quotes on the value are stripped.
func EnvValue(content []byte, key string) (string, bool) {
	var (
		value string
		found bool
	)

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		rest, ok := strings.CutPrefix(line, "ENV ")
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)

		var k, v string
		if idx := strings.IndexByte(rest, '='); idx >= 0 {
			k = strings.TrimSpace(rest[:idx])
			v = strings.TrimSpace(rest[idx+1:])
		} else if fields := strings.SplitN(rest, " ", 2); len(fields) == 2 {
			k = strings.TrimSpace(fields[0])
			v = strings.TrimSpace(fields[1])
		} else {
			continue
		}

		if k != key {
			continue
		}
		value = unquote(v)
		found = true
	}

	return value, found
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
