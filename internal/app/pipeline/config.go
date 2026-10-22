package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Galdoba/lazyam3/internal/domain/callname"
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/domain/script"
)

// defaultPriority is written into generated scripts until config (task 10)
// exposes Environment.Priority.
const defaultPriority = 6

// buildScriptConfig assembles the domain ScriptConfig for one unit.
// Environment paths are read from executorPaths (executor filesystem).
// Predictions are computed against the executor's target path so that
// ffmpeg output paths are accessible to the executor.
// The script destination is named by OUTBASE extracted from predictions.
func buildScriptConfig(unit ingest.MediaUnit, prod metadata.Product, root string, executorPaths map[string]string, hardsubTag string) (script.ScriptConfig, error) {
	if root == "" {
		return script.ScriptConfig{}, fmt.Errorf("pipeline: destination root is required")
	}
	if executorPaths == nil {
		executorPaths = map[string]string{}
	}

	streams := make([]ingest.Stream, 0, len(unit.Streams))
	for _, s := range unit.Streams {
		streams = append(streams, s)
	}
	slices.SortFunc(streams, func(a, b ingest.Stream) int {
		if a.FileIndex != b.FileIndex {
			return a.FileIndex - b.FileIndex
		}
		return a.Index - b.Index
	})

	// Use executor's target path for predictions so paths are valid on executor.
	// TOML keys: buffer, in_progress, done, target, archive, notifications
	targetRoot := executorPaths["target"]
	if targetRoot == "" {
		targetRoot = filepath.Join(root, "target")
	}
	// Normalize paths for Windows (backslash → forward slash)
	targetRoot = toSlash(targetRoot)

	// Predict outputs from the full MediaUnit — this is the single source
	// of truth for output file names. Called before the script domain is
	// touched, so predictions drive all downstream logic.
	predictions := callname.PredictOutputs(prod, unit, targetRoot, hardsubTag)

	// Build call names for all unit files (original filename → call name).
	timestamp := time.Now().Format("060102150405")
	callNames, err := callname.BuildInputNames(prod, unit.Filenames, timestamp)
	if err != nil {
		return script.ScriptConfig{}, fmt.Errorf("build call names for unit %d: %w", unit.ID, err)
	}

	// Derive SRT from original filenames, HSUBFile from predictions.
	srt, _ := findSubtitle(unit.Filenames, hardsubTag)
	hsubFile := ""
	for _, p := range predictions {
		if p.MediaType == "video" {
			hsubFile = p.HsubFile
			break
		}
	}

	// Resolve SRT and HSUB file paths to their call names.
	srtCall := callNames[srt]
	hsubCall := callNames[hsubFile]

	// Extract OUTBASE from first video prediction for script filename.
	outBase := extractOutBase(predictions)

	// Build destination path using OUTBASE instead of unit ID.
	destination := filepath.Join(root, fmt.Sprintf("%s.sh", outBase))

	// Normalize all executor paths for Windows
	buffer := toSlash(executorPaths["buffer"])
	if buffer == "" {
		buffer = toSlash(root)
	}
	inProgress := toSlash(executorPaths["in_progress"])
	if inProgress == "" {
		inProgress = toSlash(root)
	}
	done := toSlash(executorPaths["done"])
	if done == "" {
		done = toSlash(root)
	}
	notifications := toSlash(executorPaths["notifications"])
	if notifications == "" {
		notifications = toSlash(root)
	}
	archive := toSlash(executorPaths["archive"])
	if archive == "" {
		archive = toSlash(filepath.Join(root, "archive"))
	}

	return script.ScriptConfig{
		Filenames:   unit.Filenames,
		Streams:     streams,
		SRT:         srtCall,
		HSUBFile:    hsubCall,
		Product:     prod,
		Predictions: predictions,
		CallNames:   callNames,
		Env: script.Environment{
			Buffer:        buffer,
			InProgress:    inProgress,
			Done:          done,
			Notifications: notifications,
			Archive:       archive,
			TargetRoot:    targetRoot,
			Priority:      defaultPriority,
		},
		Destination: destination,
	}, nil
}

// toSlash normalizes a path to use forward slashes (Windows → Unix).
func toSlash(path string) string {
	if path == "" {
		return ""
	}
	return filepath.ToSlash(path)
}

// extractOutBase derives the OUTBASE (filename stem before _HD) from predictions.
// Returns the base filename without extension and without _HD suffix.
func extractOutBase(predictions []callname.PredictedOutput) string {
	if len(predictions) == 0 {
		return ""
	}
	// Find first video prediction
	var path string
	for _, p := range predictions {
		if p.MediaType == "video" {
			path = p.Path
			break
		}
	}
	if path == "" {
		path = predictions[0].Path
	}

	base := filepath.Base(path) // "Show_s01e06_HD.mp4"
	ext := filepath.Ext(base)   // ".mp4"
	base = base[:len(base)-len(ext)]
	if strings.HasSuffix(base, "_HD") {
		base = base[:len(base)-3] // "Show_s01e06"
	}
	return base
}

// EnsureEnvDirs creates the work-directory tree under the output root that
// generated scripts assume to exist: buffer, in_progress, done (plus done/bash
// for script self-archiving), notifications, target, and scripts.
// Layout is defined here next to buildScriptConfig — keep them in sync.
// NOTE: This function is kept for backward compatibility but is no longer
// called by the pipeline when executor paths are configured.
func EnsureEnvDirs(root string) error {
	if root == "" {
		return fmt.Errorf("pipeline: destination root is required")
	}
	for _, dir := range []string{
		"buffer",
		"in_progress",
		"done/bash",
		"notifications",
		"target",
		"scripts",
	} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return fmt.Errorf("create env dir %q: %w", dir, err)
		}
	}
	return nil
}

// findSubtitle splits subtitle filenames into the external subtitle (SRT,
// copied to target) and the hardsub file (HSUBFile, burned in). A sidecar
// whose name contains hardsubTag (case-insensitive) is treated as a hardsub;
// empty tag disables detection (all subtitles become SRT). First match in
// unit file order wins for each role.
func findSubtitle(filenames []string, hardsubTag string) (srt, hsub string) {
	for _, f := range filenames {
		if !isSubtitle(f) {
			continue
		}
		if hardsubTag != "" && strings.Contains(strings.ToLower(f), strings.ToLower(hardsubTag)) {
			if hsub == "" {
				hsub = f
			}
			continue
		}
		if srt == "" {
			srt = f
		}
	}
	return srt, hsub
}

// isSubtitle returns true when name ends with a known subtitle extension.
func isSubtitle(name string) bool {
	l := strings.ToLower(name)
	return strings.HasSuffix(l, ".srt") || strings.HasSuffix(l, ".ass") || strings.HasSuffix(l, ".ssa")
}
