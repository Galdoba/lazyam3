package ingest

import (
	"fmt"
)

// ExtractID parses the product ID from a filename stem (name without extension).
// The stem must start with a digit sequence — that sequence is the ID.
// Returns an error if the stem does not start with a digit.
func ExtractID(filename string) (int64, error) {
	name := filename
	if idx := len(filename) - 1; idx >= 0 {
		for idx >= 0 && filename[idx] != '.' {
			idx--
		}
		if idx >= 0 {
			name = filename[:idx]
		}
	}

	if len(name) == 0 || name[0] < '0' || name[0] > '9' {
		return 0, fmt.Errorf("filename %q does not start with a digit", filename)
	}

	var digits string
	for _, r := range name {
		if r >= '0' && r <= '9' {
			digits += string(r)
		} else {
			break
		}
	}

	var id int64
	fmt.Sscanf(digits, "%d", &id) //nolint:errcheck // best-effort
	return id, nil
}
