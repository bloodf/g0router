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

	call.token, call.err = refresh(ctx, conn)
	if call.err != nil {
		call.err = fmt.Errorf("refresh token: %w", call.err)
	}

	m.mu.Lock()
	delete(m.inflight, key)
	m.mu.Unlock()
	close(call.done)

	return call.token, call.err
}

func refreshKey(conn *store.Connection) string {
	token := ""
	if conn.RefreshToken != nil {
		token = *conn.RefreshToken
	}
	return conn.Provider + "\x00" + conn.ID + "\x00" + token
}
