package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
)

var ErrRefreshConnectionRequired = errors.New("provider: refresh connection required")

type TokenRefreshFunc func(context.Context, *store.Connection) (oauth.TokenResult, error)

type RefreshManager struct {
	mu       sync.Mutex
	inflight map[string]*refreshCall
}

type refreshCall struct {
	done  chan struct{}
	token oauth.TokenResult
	err   error
}

func NewRefreshManager() *RefreshManager {
	return &RefreshManager{
		inflight: make(map[string]*refreshCall),
	}
}

func (m *RefreshManager) Refresh(ctx context.Context, conn *store.Connection, refresh TokenRefreshFunc) (oauth.TokenResult, error) {
	if conn == nil {
		return oauth.TokenResult{}, ErrRefreshConnectionRequired
	}

	key := refreshKey(conn)

	m.mu.Lock()
	if call, ok := m.inflight[key]; ok {
		m.mu.Unlock()
		<-call.done
		return call.token, call.err
	}
	call := &refreshCall{done: make(chan struct{})}
	m.inflight[key] = call
	m.mu.Unlock()

	call.token, call.err = safeRefresh(ctx, conn, refresh)
	if call.err != nil {
		call.err = fmt.Errorf("refresh token: %w", call.err)
	}

	// Unblock waiters before removing the inflight entry so a new caller can
	// never observe a deleted entry while this call's result is still pending.
	close(call.done)
	m.mu.Lock()
	delete(m.inflight, key)
	m.mu.Unlock()

	return call.token, call.err
}

// safeRefresh runs refresh and converts a panic into an error so single-flight
// waiters are never left blocked on a never-closed done channel.
func safeRefresh(ctx context.Context, conn *store.Connection, refresh TokenRefreshFunc) (token oauth.TokenResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("refresh panicked: %v", r)
		}
	}()
	return refresh(ctx, conn)
}

func refreshKey(conn *store.Connection) string {
	token := ""
	if conn.RefreshToken != nil {
		token = *conn.RefreshToken
	}
	return conn.Provider + "\x00" + conn.ID + "\x00" + token
}
