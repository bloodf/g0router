package inference

import (
	"errors"
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/store"
)

// AccountRunner is the production ModelRunner implementation.
// It delegates account-level fallback to a SelectionEngine and wraps HTTP
// 502/503/504 failures with ErrModelTransient so combo fallback can apply a
// short cooldown before trying the next model.
type AccountRunner struct {
	sel       *SelectionEngine
	projectID *ProjectIDManager // optional, additive (PAR-ROUTE-053); nil = no-op
}

// NewAccountRunner creates an AccountRunner backed by the given selection engine.
func NewAccountRunner(sel *SelectionEngine) *AccountRunner {
	return &AccountRunner{sel: sel}
}

// SetProjectIDManager wires the optional project-ID cold-miss manager
// (PAR-ROUTE-053). Additive: NewAccountRunner's signature is unchanged; when no
// manager is set, EnsureProjectID is a no-op and the run path is identical.
func (a *AccountRunner) SetProjectIDManager(m *ProjectIDManager) {
	a.projectID = m
}

// RunModel resolves the model to a provider and executes fn against successive
// connections via the selection engine. On final failure, if the underlying
// error is a ProviderError with status 502/503/504, the returned error joins
// ErrModelTransient so the caller can treat it as transient.
func (a *AccountRunner) RunModel(model string, fn func(*store.Connection) (Verdict, error)) error {
	providerID, ok := providerForModel(model)
	if !ok {
		return fmt.Errorf("no provider for model %s", model)
	}

	var lastErr error
	err := a.sel.WithAccountFallback(providerID, model, func(conn *store.Connection) (Verdict, error) {
		// PAR-ROUTE-053: resolve+persist the project ID on a cold miss for
		// providers that need it (antigravity/gemini-cli) before dispatch.
		// Additive: a nil manager is a no-op. A resolution failure is treated
		// as a permanent verdict for this connection's attempt.
		if _, pErr := a.projectID.EnsureProjectID(conn); pErr != nil {
			lastErr = pErr
			return VerdictPermanent, pErr
		}
		verdict, fnErr := fn(conn)
		if fnErr != nil {
			lastErr = fnErr
		}
		return verdict, fnErr
	})

	if err == nil {
		return nil
	}

	checkErr := lastErr
	if checkErr == nil {
		checkErr = err
	}

	var pe *schemas.ProviderError
	if errors.As(checkErr, &pe) && isTransientStatus(pe.StatusCode) {
		if lastErr != nil {
			return errors.Join(ErrModelTransient, lastErr)
		}
		return errors.Join(ErrModelTransient, err)
	}

	if lastErr != nil {
		return lastErr
	}
	return err
}

// ModelRetryAfter returns the earliest retry-after time for the model across
// all accounts, delegating to the cooldown engine.
func (a *AccountRunner) ModelRetryAfter(model string, now time.Time) (time.Time, bool, error) {
	providerID, ok := providerForModel(model)
	if !ok {
		return time.Time{}, false, fmt.Errorf("no provider for model %s", model)
	}
	return a.sel.cd.GroupRetryAfter(providerID, model, now)
}

func isTransientStatus(code int) bool {
	return code == 502 || code == 503 || code == 504
}
