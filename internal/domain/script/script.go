package script

// StreamSelection represents a stream selection rule for the rendering script.
type StreamSelection struct {
	InputIndex  int
	StreamIndex int
	Kind        string // "v", "a", "s"
}

// Script is a rendered bash script ready to be written to disk.
type Script struct {
	Destination string
	Content     string
}
