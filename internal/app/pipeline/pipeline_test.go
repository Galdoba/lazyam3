package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/domain/script"
)

type fakeOrganizer struct {
	units map[string]ingest.MediaUnit
	errs  map[string]error
}

func (f *fakeOrganizer) Ingest(dir string) (ingest.MediaUnit, error) {
	if err, ok := f.errs[dir]; ok {
		return ingest.MediaUnit{}, err
	}
	if u, ok := f.units[dir]; ok {
		return u, nil
	}
	return ingest.MediaUnit{}, fmt.Errorf("unexpected dir %q", dir)
}

type fakeRepo struct {
	products map[int64]*metadata.Product
	errs     map[int64]error
}

func (f *fakeRepo) FindByID(id int64) (*metadata.Product, error) {
	if err, ok := f.errs[id]; ok {
		return nil, err
	}
	if p, ok := f.products[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("product %d not found", id)
}

type fakeRenderer struct {
	scripts []script.Script
	err     error
}

func (f *fakeRenderer) Render(sc script.Script) error {
	f.scripts = append(f.scripts, sc)
	return f.err
}

func testUnit() ingest.MediaUnit {
	return ingest.MediaUnit{
		ID:        12094,
		Filenames: []string{"12094.mp4", "12094-rus.srt"},
		Streams: map[string]ingest.Stream{
			"[1:s:0]": {FileIndex: 1, Index: 0, Kind: ingest.StreamKindSubtitle},
			"[0:a:1]": {FileIndex: 0, Index: 1, Kind: ingest.StreamKindAudio},
			"[0:v:0]": {FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo},
		},
	}
}

func testProduct() *metadata.Product {
	season, episode := 1, 6
	return &metadata.Product{
		ID:            12094,
		Kind:          metadata.KindSeriesEpisode,
		SeriesTitle:   "На грани",
		SeasonNumber:  &season,
		EpisodeNumber: &episode,
	}
}

// sourceRoot creates a scan fixture: <root>/ep1/12094.mp4 (writable)
// and returns (root, unitDir).
func sourceRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "ep1")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "12094.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, dir
}

func newTestPipeline(t *testing.T, root string, in *fakeOrganizer, repo *fakeRepo, rd *fakeRenderer, events chan PipelineEvent, dryRun bool) *Pipeline {
	t.Helper()
	return NewPipeline(Options{
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Ingest:       in,
		Metadata:     repo,
		Renderer:     rd,
		Events:       events,
		SourceDir:    root,
		Destination:  t.TempDir(),
		DryRun:       dryRun,
		HardsubTag:   "",
	})
}

func drainEvents(ch chan PipelineEvent) []PipelineEvent {
	var out []PipelineEvent
	for {
		select {
		case e := <-ch:
			out = append(out, e)
		default:
			return out
		}
	}
}

func TestRunOnce_ProcessesUnit(t *testing.T) {
	root, unitDir := sourceRoot(t)
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, false)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Processed != 1 || out.Failed != 0 {
		t.Errorf("RunOnce() = %+v, want Processed=1 Failed=0", out)
	}

	if len(rd.scripts) != 1 {
		t.Fatalf("renderer got %d scripts, want 1", len(rd.scripts))
	}
	sc := rd.scripts[0]
	if !strings.HasSuffix(sc.Destination, filepath.Join("scripts", "12094.sh")) {
		t.Errorf("Destination = %q, want .../scripts/12094.sh", sc.Destination)
	}
	if !strings.HasPrefix(sc.Content, "#!/bin/bash") {
		t.Errorf("Content does not start with shebang")
	}

	evs := drainEvents(events)
	if len(evs) != 2 || evs[0].Type != EventUnitDone || evs[0].ID != 12094 || evs[1].Type != EventScanDone {
		t.Errorf("events = %+v, want [UnitDone(12094) ScanDone]", evs)
	}
}

func TestRunOnce_MetadataError(t *testing.T) {
	root, unitDir := sourceRoot(t)
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{errs: map[int64]error{12094: errors.New("not found")}},
		rd, events, false)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Processed != 0 || out.Failed != 1 {
		t.Errorf("RunOnce() = %+v, want Processed=0 Failed=1", out)
	}
	if len(rd.scripts) != 0 {
		t.Errorf("renderer should not be called on metadata failure")
	}

	evs := drainEvents(events)
	if len(evs) != 2 || evs[0].Type != EventUnitFailed || evs[0].ID != 12094 || evs[1].Type != EventScanDone {
		t.Errorf("events = %+v, want [UnitFailed(12094) ScanDone]", evs)
	}
}

func TestRunOnce_IngestError(t *testing.T) {
	root, unitDir := sourceRoot(t)
	events := make(chan PipelineEvent, 100)
	p := newTestPipeline(t, root,
		&fakeOrganizer{errs: map[string]error{unitDir: errors.New("probe failed")}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		&fakeRenderer{}, events, false)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Failed != 1 {
		t.Errorf("Failed = %d, want 1", out.Failed)
	}
	evs := drainEvents(events)
	if len(evs) != 2 || evs[0].Type != EventUnitFailed || evs[0].ID != 0 {
		t.Errorf("events = %+v, want [UnitFailed(0) ScanDone]", evs)
	}
}

func TestRunOnce_DryRunSkipsBuild(t *testing.T) {
	root, unitDir := sourceRoot(t)
	events := make(chan PipelineEvent, 100)
	rd := &fakeRenderer{}
	p := newTestPipeline(t, root,
		&fakeOrganizer{units: map[string]ingest.MediaUnit{unitDir: testUnit()}},
		&fakeRepo{products: map[int64]*metadata.Product{12094: testProduct()}},
		rd, events, true)

	out, err := p.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error: %v", err)
	}
	if out.Processed != 1 || out.Failed != 0 {
		t.Errorf("RunOnce() = %+v, want Processed=1 Failed=0", out)
	}
	if len(rd.scripts) != 0 {
		t.Errorf("dry run must not render scripts, got %d", len(rd.scripts))
	}
}

func TestRunOnce_ScanError(t *testing.T) {
	p := NewPipeline(Options{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Ingest:    &fakeOrganizer{},
		Metadata:  &fakeRepo{},
		Renderer:  &fakeRenderer{},
		SourceDir: filepath.Join(t.TempDir(), "does-not-exist"),
	})
	if _, err := p.RunOnce(context.Background()); err == nil {
		t.Fatal("RunOnce() on missing source dir: expected error, got nil")
	}
}

func TestRun_CtxCancelled(t *testing.T) {
	p := newTestPipeline(t, t.TempDir(), &fakeOrganizer{}, &fakeRepo{}, &fakeRenderer{}, nil, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}

func TestBuildScriptConfig(t *testing.T) {
	unit := testUnit()
	root := t.TempDir()
	cfg, err := buildScriptConfig(unit, *testProduct(), root, "")
	if err != nil {
		t.Fatalf("buildScriptConfig() error: %v", err)
	}

	if cfg.SRT != "12094-rus.srt" {
		t.Errorf("SRT = %q, want 12094-rus.srt", cfg.SRT)
	}
	if cfg.Destination != filepath.Join(root, "scripts", "12094.sh") {
		t.Errorf("Destination = %q", cfg.Destination)
	}
	if cfg.Env.Buffer != filepath.Join(root, "buffer") ||
		cfg.Env.InProgress != filepath.Join(root, "in_progress") ||
		cfg.Env.Done != filepath.Join(root, "done") ||
		cfg.Env.Notifications != filepath.Join(root, "notifications") ||
		cfg.Env.TargetRoot != filepath.Join(root, "target") {
		t.Errorf("Env paths not derived from root: %+v", cfg.Env)
	}

	// Streams sorted by (FileIndex, Index) regardless of map order.
	want := []ingest.Stream{
		{FileIndex: 0, Index: 0, Kind: ingest.StreamKindVideo},
		{FileIndex: 0, Index: 1, Kind: ingest.StreamKindAudio},
		{FileIndex: 1, Index: 0, Kind: ingest.StreamKindSubtitle},
	}
	if len(cfg.Streams) != len(want) {
		t.Fatalf("Streams len = %d, want %d", len(cfg.Streams), len(want))
	}
	for i, w := range want {
		if cfg.Streams[i].FileIndex != w.FileIndex || cfg.Streams[i].Index != w.Index || cfg.Streams[i].Kind != w.Kind {
			t.Errorf("Streams[%d] = %+v, want %+v", i, cfg.Streams[i], w)
		}
	}
}

func TestBuildScriptConfig_RequiresRoot(t *testing.T) {
	if _, err := buildScriptConfig(testUnit(), *testProduct(), "", ""); err == nil {
		t.Fatal("expected error for empty destination root, got nil")
	}
}

func TestEnsureEnvDirs(t *testing.T) {
	root := t.TempDir()
	if err := EnsureEnvDirs(root); err != nil {
		t.Fatalf("EnsureEnvDirs() error: %v", err)
	}
	for _, dir := range []string{"buffer", "in_progress", "done/bash", "notifications", "target", "scripts"} {
		if fi, err := os.Stat(filepath.Join(root, dir)); err != nil || !fi.IsDir() {
			t.Errorf("env dir %q missing or not a directory (err=%v)", dir, err)
		}
	}
	if err := EnsureEnvDirs(""); err == nil {
		t.Error("EnsureEnvDirs(\"\"): expected error, got nil")
	}
}

func TestFindSRT(t *testing.T) {
	tests := []struct {
		names      []string
		hardsubTag string
		wantSrt    string
		wantHsub   string
	}{
		{[]string{"a.mp4", "a.srt"}, "", "a.srt", ""},
		{[]string{"a.mp4", "a.srt"}, "hardsub", "a.srt", ""},
		{[]string{"a.mp4", "a_hardsub.srt"}, "", "a_hardsub.srt", ""},
		{[]string{"a.mp4", "a_hardsub.srt"}, "hardsub", "", "a_hardsub.srt"},
		{[]string{"a.mp4"}, "", "", ""},
		{nil, "", "", ""},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v-%s", tt.names, tt.hardsubTag), func(t *testing.T) {
			gotSrt, gotHsub := findSubtitle(tt.names, tt.hardsubTag)
			if gotSrt != tt.wantSrt {
				t.Errorf("findSubtitle(%v, %q) srt = %q, want %q", tt.names, tt.hardsubTag, gotSrt, tt.wantSrt)
			}
			if gotHsub != tt.wantHsub {
				t.Errorf("findSubtitle(%v, %q) hsub = %q, want %q", tt.names, tt.hardsubTag, gotHsub, tt.wantHsub)
			}
		})
	}
}
