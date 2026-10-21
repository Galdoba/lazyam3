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
const defaultPriority = 0

// buildScriptConfig assembles the domain ScriptConfig for one unit.
// Work-directory environment paths are derived by convention under the
// output root: <root>/{buffer,in_progress,done,notifications,target};
// the script artifact goes to <root>/scripts/<id>.sh.
// Predictions are the single source of truth for output names, computed
// from the full MediaUnit before the script domain is touched.
func buildScriptConfig(unit ingest.MediaUnit, prod metadata.Product, root string, hardsubTag string) (script.ScriptConfig, error) {
	if root == "" {
		return script.ScriptConfig{}, fmt.Errorf("pipeline: destination root is required")
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

	// Predict outputs from the full MediaUnit — this is the single source
	// of truth for output file names. Called before the script domain is
	// touched, so predictions drive all downstream logic.
	predictions := callname.PredictOutputs(prod, unit, filepath.Join(root, "target"), hardsubTag)

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

	return script.ScriptConfig{
		Filenames:   unit.Filenames,
		Streams:     streams,
		SRT:         srtCall,
		HSUBFile:    hsubCall,
		Product:     prod,
		Predictions: predictions,
		CallNames:   callNames,
		Env: script.Environment{
			Buffer:        filepath.Join(root, "buffer"),
			InProgress:    filepath.Join(root, "in_progress"),
			Done:          filepath.Join(root, "done"),
			Notifications: filepath.Join(root, "notifications"),
			TargetRoot:    filepath.Join(root, "target"),
			Priority:      defaultPriority,
		},
		Destination: filepath.Join(root, "scripts", fmt.Sprintf("%d.sh", unit.ID)),
	}, nil
}

// EnsureEnvDirs creates the work-directory tree under the output root that
// generated scripts assume to exist: buffer, in_progress, done (plus done/bash
// for script self-archiving), notifications, target, and scripts.
// Layout is defined here next to buildScriptConfig — keep them in sync.
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
