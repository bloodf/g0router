package store

import (
	"strings"
	"testing"
)

func TestClampAuditLimitBounds(t *testing.T) {
	if got := clampAuditLimit(0); got != defaultAuditLimit {
		t.Errorf("clampAuditLimit(0) = %d, want %d", got, defaultAuditLimit)
	}
	if got := clampAuditLimit(-5); got != defaultAuditLimit {
		t.Errorf("clampAuditLimit(-5) = %d, want %d", got, defaultAuditLimit)
	}
	if got := clampAuditLimit(maxAuditLimit + 100); got != maxAuditLimit {
		t.Errorf("clampAuditLimit(over-max) = %d, want %d", got, maxAuditLimit)
	}
	if got := clampAuditLimit(10); got != 10 {
		t.Errorf("clampAuditLimit(10) = %d, want 10", got)
	}
}

func TestAppendAuditClosedStoreErrors(t *testing.T) {
	s := newAuditStore(t)
	_ = s.Close()
	err := s.AppendAudit(AuditEntry{Action: "POST /api/keys", Target: "x", Details: "d"})
	if err == nil {
		t.Fatal("expected error appending to closed store")
	}
	if !strings.Contains(err.Error(), "insert audit log") {
		t.Fatalf("error = %v, want wrapped insert audit log", err)
	}
}

func TestListAuditClosedStoreErrors(t *testing.T) {
	s := newAuditStore(t)
	_ = s.Close()
	_, _, err := s.ListAudit(AuditFilter{})
	if err == nil {
		t.Fatal("expected error listing from closed store")
	}
	if !strings.Contains(err.Error(), "count audit log") {
		t.Fatalf("error = %v, want wrapped count audit log", err)
	}
}
