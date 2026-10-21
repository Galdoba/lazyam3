package script

// Renderer materializes a built Script as a usable artifact
// (e.g., an executable shell file at Script.Destination).
type Renderer interface {
	Render(sc Script) error
}
