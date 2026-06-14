package governance

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// AuditService records and reads administrative audit entries.
type AuditService struct {
	store *store.Store
	now   func() time.Time
}

// NewAuditService creates an audit service backed by the store.
func NewAuditService(st *store.Store) *AuditService {
	return &AuditService{store: st, now: time.Now}
}

// List returns up to limit entries (newest first) and the total entry count.
func (a *AuditService) List(limit int) (items []store.AuditEntry, total int, err error) {
	total, err = a.store.CountAuditEntries()
	if err != nil {
		return nil, 0, err
	}
	items, err = a.store.ListAuditEntries(limit)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// WriteAudit records a single administrative action. details must be a
// human-readable summary and must never contain secrets (passwords, tokens).
func (a *AuditService) WriteAudit(actor, action, target, details string) error {
	id, err := auditID()
	if err != nil {
		return err
	}
	return a.store.InsertAuditEntry(store.AuditEntry{
		ID:        id,
		Timestamp: a.now().UTC().Format(time.RFC3339Nano),
		Actor:     actor,
		Action:    action,
		Target:    target,
		Details:   details,
	})
}

func auditID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate audit id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
