package pipeline

// RunInput is the DTO passed from presentation to the pipeline orchestrator.
type RunInput struct {
	SourceDir   string
	Destination string
	DryRun      bool
}

// RunOutput is the DTO returned from the pipeline orchestrator to presentation.
type RunOutput struct {
	FilesProcessed int
	ScriptWritten  string
}
