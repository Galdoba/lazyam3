package ingest

import (
	"fmt"
	"strings"
)

// ExtractID parses the product ID from a filename stem (name without extension).
// The stem must start with a digit sequence — that sequence is the ID.
// When the stem does not start with a digit, returns SyntheticID (9999).
func ExtractID(filename string) (int64, error) {
	name := strings.SplitN(filename, ".", 2)[0]

	if len(name) == 0 || name[0] < '0' || name[0] > '9' {
		return SyntheticID, nil
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
