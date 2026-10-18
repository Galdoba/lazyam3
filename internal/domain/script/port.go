package script

// Renderer writes a script's Text to a destination path.
type Renderer interface {
	Write(dest string, content Text) error
}
