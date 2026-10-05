package internal

import "testing"

func TestHardcoverDiagnosticsEnabledOnlyForConfiguredAuthors(t *testing.T) {
	diagnostics := newHardcoverDiagnostics("154441, 42")

	if !diagnostics.enabledFor(154441) {
		t.Fatal("configured Hardcover author must enable diagnostics")
	}
	if !diagnostics.enabledFor(42) {
		t.Fatal("each configured Hardcover author must enable diagnostics")
	}
	if diagnostics.enabledFor(7) {
		t.Fatal("unconfigured Hardcover author must not enable diagnostics")
	}
}

func TestHardcoverDiagnosticRefreshCountsStagesAndOutcomes(t *testing.T) {
	diagnostics := newHardcoverDiagnostics("154441")
	refresh := diagnostics.newRefresh(154441)

	refresh.record("source_candidate", "selected")
	refresh.record("source_candidate", "skipped_no_edition")
	refresh.record("denormalize", "inserted")

	if got := refresh.stages["source_candidate"]; got != 2 {
		t.Fatalf("source candidate count = %d, want 2", got)
	}
	if got := refresh.outcomes["selected"]; got != 1 {
		t.Fatalf("selected count = %d, want 1", got)
	}
	if got := refresh.outcomes["skipped_no_edition"]; got != 1 {
		t.Fatalf("skipped_no_edition count = %d, want 1", got)
	}
	if got := refresh.outcomes["inserted"]; got != 1 {
		t.Fatalf("inserted count = %d, want 1", got)
	}
}

func TestHardcoverDiagnosticRefreshContextIsAuthorScoped(t *testing.T) {
	diagnostics := newHardcoverDiagnostics("154441")
	ctx, refresh := diagnostics.withRefresh(t.Context(), 154441)

	if got := diagnosticRefresh(ctx); got != refresh {
		t.Fatal("configured author refresh must be available from its context")
	}
	_, otherRefresh := diagnostics.withRefresh(t.Context(), 42)
	if otherRefresh.enabled {
		t.Fatal("unconfigured author refresh must be disabled")
	}
}

func TestNewHardcoverGetterReadsDiagnosticAuthorIDsFromEnvironment(t *testing.T) {
	t.Setenv("HARDCOVER_DIAGNOSTIC_AUTHOR_IDS", "154441")

	getter, err := NewHardcoverGetter(newMemoryCache(), nil)
	if err != nil {
		t.Fatalf("NewHardcoverGetter() error = %v", err)
	}
	if !getter.diagnostics.enabledFor(154441) {
		t.Fatal("getter must enable diagnostics for configured author")
	}
	if getter.diagnostics.enabledFor(42) {
		t.Fatal("getter must not enable diagnostics for other authors")
	}
}

func TestNewControllerReadsDiagnosticAuthorIDsFromEnvironment(t *testing.T) {
	t.Setenv("HARDCOVER_DIAGNOSTIC_AUTHOR_IDS", "154441")

	controller, err := NewController(newMemoryCache(), NewMockgetter(nil), nil, nil)
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if !controller.diagnostics.enabledFor(154441) {
		t.Fatal("controller must enable diagnostics for configured author")
	}
}

func TestHardcoverDiagnosticsIgnoreEmptyAndInvalidAuthorIDs(t *testing.T) {
	diagnostics := newHardcoverDiagnostics(" , invalid, 154441, 0 ")

	if !diagnostics.enabledFor(154441) {
		t.Fatal("valid author IDs must be retained")
	}
	if diagnostics.enabledFor(0) {
		t.Fatal("zero author ID must not enable diagnostics")
	}
	if diagnostics.enabledFor(1) {
		t.Fatal("invalid author IDs must not enable diagnostics")
	}
}
