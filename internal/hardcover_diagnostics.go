package internal

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// hardcoverDiagnostics limits diagnostic logging to explicitly configured
// Hardcover author IDs. It deliberately retains only numeric IDs so diagnostic
// configuration cannot expose secrets in logs.
type hardcoverDiagnostics struct {
	authorIDs map[int64]struct{}
}

func newHardcoverDiagnostics(rawAuthorIDs string) hardcoverDiagnostics {
	diagnostics := hardcoverDiagnostics{authorIDs: make(map[int64]struct{})}
	for _, rawID := range strings.Split(rawAuthorIDs, ",") {
		authorID, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
		if err != nil || authorID <= 0 {
			continue
		}
		diagnostics.authorIDs[authorID] = struct{}{}
	}
	return diagnostics
}

func (d hardcoverDiagnostics) enabledFor(authorID int64) bool {
	_, enabled := d.authorIDs[authorID]
	return enabled
}

type hardcoverDiagnosticContextKey struct{}

func (d hardcoverDiagnostics) withRefresh(ctx context.Context, authorID int64) (context.Context, *hardcoverDiagnosticRefresh) {
	refresh := d.newRefresh(authorID)
	if !refresh.enabled {
		return ctx, refresh
	}
	return context.WithValue(ctx, hardcoverDiagnosticContextKey{}, refresh), refresh
}

func diagnosticRefresh(ctx context.Context) *hardcoverDiagnosticRefresh {
	refresh, _ := ctx.Value(hardcoverDiagnosticContextKey{}).(*hardcoverDiagnosticRefresh)
	return refresh
}

func (d hardcoverDiagnostics) newRefresh(authorID int64) *hardcoverDiagnosticRefresh {
	return &hardcoverDiagnosticRefresh{
		enabled:   d.enabledFor(authorID),
		authorID:  authorID,
		refreshID: uuid.NewString(),
		stages:    make(map[string]int),
		outcomes:  make(map[string]int),
	}
}

// hardcoverDiagnosticRefresh collects only aggregate counts in memory. Event
// data is emitted through the existing structured logger at the call site.
type hardcoverDiagnosticRefresh struct {
	enabled   bool
	authorID  int64
	refreshID string

	mu       sync.Mutex
	stages   map[string]int
	outcomes map[string]int
}

func (r *hardcoverDiagnosticRefresh) record(stage, outcome string) {
	if !r.enabled {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stages[stage]++
	r.outcomes[outcome]++
}

func (r *hardcoverDiagnosticRefresh) event(ctx context.Context, stage, outcome string, attrs ...slog.Attr) {
	if !r.enabled {
		return
	}
	r.record(stage, outcome)
	attrs = append([]slog.Attr{
		slog.String("diagnostic", "hardcover_bibliography"),
		slog.String("refreshID", r.refreshID),
		slog.Int64("authorID", r.authorID),
		slog.String("stage", stage),
		slog.String("outcome", outcome),
	}, attrs...)
	Log(ctx).LogAttrs(ctx, slog.LevelInfo, "hardcover bibliography diagnostic", attrs...)
}

func (r *hardcoverDiagnosticRefresh) summary(ctx context.Context, reason string) {
	if !r.enabled {
		return
	}
	r.mu.Lock()
	stages := make(map[string]int, len(r.stages))
	for stage, count := range r.stages {
		stages[stage] = count
	}
	outcomes := make(map[string]int, len(r.outcomes))
	for outcome, count := range r.outcomes {
		outcomes[outcome] = count
	}
	r.mu.Unlock()

	Log(ctx).Info("hardcover bibliography diagnostic summary",
		"diagnostic", "hardcover_bibliography",
		"refreshID", r.refreshID,
		"authorID", r.authorID,
		"reason", reason,
		"stages", stages,
		"outcomes", outcomes,
	)
}
