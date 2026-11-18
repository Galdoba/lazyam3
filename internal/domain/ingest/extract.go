package ingest

import (
	"strconv"
)

// extractID decides the product identity that a set of candidate file names
// claims. It never returns an error: a name either claims an ID or claims
// nothing.
//
// Accepted name form is "<digits> [ '-' tag [ '_' tag ]* ] [ '.' extension ]".
// The leading digit run must end at '-', '.' or the end of the name; digits
// followed directly by any other character make the name claim nothing.
// A claim is a positive number, so a literal "0" stem claims nothing and 0 is
// reserved for "no identity".
//
// The result is the unanimous claim: every candidate name must claim the same
// ID. If any candidate claims nothing, or the candidates claim different IDs,
// the result is 0 and no candidate's claim is adopted on its own.
//
// Candidate names are filtered before the call. Extraction only answers whether
// the names it was given agree; it never decides who is asked.
func extractID(candidateNames ...string) int64 {
	var id int64
	for i, name := range candidateNames {
		claim := claimedID(name)
		if claim == 0 {
			return 0
		}
		if i == 0 {
			id = claim
			continue
		}
		if claim != id {
			return 0
		}
	}
	return id
}

// claimedID returns the product ID a single file name claims, or 0 when it
// claims none.
func claimedID(filename string) int64 {
	end := 0
	for end < len(filename) && filename[end] >= '0' && filename[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	if end < len(filename) && filename[end] != '-' && filename[end] != '.' {
		return 0
	}

	id, err := strconv.ParseInt(filename[:end], 10, 64)
	if err != nil {
		return 0
	}
	return id
}
