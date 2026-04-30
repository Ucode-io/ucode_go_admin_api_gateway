package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

type SSEEventType = string

const (
	EvProgress   SSEEventType = "progress"
	EvPlan       SSEEventType = "plan"
	EvTableStart SSEEventType = "table_start"
	EvTableDone  SSEEventType = "table_done"
	EvManifest   SSEEventType = "manifest"
	EvChunkStart SSEEventType = "chunk_start"
	EvChunkDone  SSEEventType = "chunk_done"
	EvRepair     SSEEventType = "repair"
	EvPublish    SSEEventType = "publish"
	EvDone       SSEEventType = "done"
	EvError      SSEEventType = "error"
)

// SSEEvent is a single server-sent event in the generation progress stream.
type SSEEvent struct {
	Type    SSEEventType `json:"type"`
	Message string       `json:"message,omitempty"`
	Percent int          `json:"percent,omitempty"`
	Data    any          `json:"data,omitempty"`
}

// PlanEventData carries architect results so the frontend can show planned pages/tables.
type PlanEventData struct {
	ProjectName string   `json:"project_name"`
	ProjectType string   `json:"project_type"`
	Tables      []string `json:"tables"`
	TableCount  int      `json:"table_count"`
}

// ManifestEventData carries manifest results: total files and feature group names.
type ManifestEventData struct {
	TotalFiles   int      `json:"total_files"`
	GroupCount   int      `json:"group_count"`
	FeatureNames []string `json:"feature_names"`
}

// ChunkDoneData carries one completed feature chunk with all its files so the
// frontend can display a live code preview as each feature is generated.
type ChunkDoneData struct {
	Feature string               `json:"feature"`
	Index   int                  `json:"index"`
	Total   int                  `json:"total"`
	Files   []models.ProjectFile `json:"files"`
}

// DoneEventData is the terminal event payload.
type DoneEventData struct {
	Description     string `json:"description"`
	MicrofrontendID string `json:"microfrontend_id,omitempty"`
	McpProjectID    string `json:"mcp_project_id,omitempty"`
	FileCount       int    `json:"file_count,omitempty"`
	DurationSec     int    `json:"duration_sec"`
}

// ProgressEmitter receives generation progress events.
// All Emit calls must be safe from multiple goroutines concurrently.
type ProgressEmitter interface {
	Emit(SSEEvent)
}

// channelEmitter writes events to a buffered channel without blocking.
// If the buffer is full the event is silently dropped — the pipeline must never stall.
type channelEmitter struct {
	ch chan<- SSEEvent
}

func (e *channelEmitter) Emit(ev SSEEvent) {
	// Protect against writes to closed channel if background tasks (e.g. async table creation)
	// outlive the main pipeline and try to emit after eventCh is closed.
	defer func() {
		if r := recover(); r != nil {
			// channel is closed, safely ignore the event
		}
	}()
	select {
	case e.ch <- ev:
	default:
	}
}

// noopEmitter discards all events.
// Used when SSE is not requested (backwards-compat with the existing JSON endpoint).
type noopEmitter struct{}

func (noopEmitter) Emit(SSEEvent) {}

func writeSSEEvent(w io.Writer, ev SSEEvent) {
	data, _ := json.Marshal(ev)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// withHeartbeat runs fn in a goroutine and emits heartbeat progress events every 12 s
// until fn completes. Percent slowly advances from pctStart toward pctEnd
// (capped at 90 % of the range so it never reaches pctEnd before fn finishes).
func withHeartbeat(ctx context.Context, emit ProgressEmitter, messages []string, pctStart, pctEnd int, estimatedDur time.Duration, fn func() error) error {
	done := make(chan error, 1)
	go func() { done <- fn() }()

	ticker := time.NewTicker(12 * time.Second)
	defer ticker.Stop()

	start := time.Now()
	msgIdx := 0

	for {
		select {
		case err := <-done:
			return err
		case <-ticker.C:
			frac := time.Since(start).Seconds() / estimatedDur.Seconds()
			if frac > 0.9 {
				frac = 0.9
			}
			pct := pctStart + int(float64(pctEnd-pctStart)*frac)
			emit.Emit(SSEEvent{Type: EvProgress, Message: messages[msgIdx%len(messages)], Percent: pct})
			msgIdx++
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
