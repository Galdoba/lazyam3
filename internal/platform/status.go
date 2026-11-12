package platform

import (
	"fmt"
	"os"
)

// StdStatusWriter writes status lines to stderr using ANSI \r to overwrite.
type StdStatusWriter struct{}

// NewStatusWriter returns a new StdStatusWriter.
func NewStatusWriter() StdStatusWriter {
	return StdStatusWriter{}
}

// Write emits a single-line status to stderr.
func (w StdStatusWriter) Write(status string) {
	fmt.Fprintf(os.Stderr, "\r%s\r", status)
}
