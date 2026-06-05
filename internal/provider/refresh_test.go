package provider

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
)

func TestRefreshManagerDedupsConcurrentRefreshForSameConnectionToken(t *testing.T) {
	manager := NewRefreshManager()
	refreshToken := "refresh-token"
	conn := &store.Connection{ID: "conn-1", Provider: "anthropic", RefreshToken: &refreshToken}
	want := oauth.TokenResult{
		Provider:     oauth.ProviderID("anthropic"),
		AccessToken:  "access-token",
		RefreshToken: "next-refresh-token",
	}
	refresher := newBlockingRefresh(want, nil)

	firstResult := make(chan oauth.TokenResult, 1)
	firstErr := make(chan error, 1)
	go func() {
		token, err := manager.Refresh(context.Background(), conn, refresher.Refresh)
		if err != nil {
			firstErr <- err
			return
		}
		firstResult <- token
	}()
	refresher.waitForCalls(t, 1)

	secondResult := make(chan oauth.TokenResult, 1)
	secondErr := make(chan error, 1)
	go func() {
		token, err := manager.Refresh(context.Background(), conn, refresher.Refresh)
		if err != nil {
			secondErr <- err
			return
		}
		secondResult <- token
	}()

	refresher.waitForStableCalls(t, 1)
	refresher.release()

	for range 2 {
		select {
		case err := <-firstErr:
			t.Fatalf("Refresh returned error: %v", err)
		case err := <-secondErr:
			t.Fatalf("Refresh returned error: %v", err)
		case got := <-firstResult:
			assertTokenResult(t, got, want)
		case got := <-secondResult:
			assertTokenResult(t, got, want)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for refresh result")
		}
	}

	if calls := refresher.calls(); calls != 1 {
		t.Fatalf("refresh calls = %d, want 1", calls)
	}
}

func TestRefreshManagerDoesNotDedupDifferentRefreshTokens(t *testing.T) {
	manager := NewRefreshManager()
	firstToken := "refresh-token-1"
	secondToken := "refresh-token-2"
	firstConn := &store.Connection{ID: "conn-1", Provider: "anthropic", RefreshToken: &firstToken}
	secondConn := &store.Connection{ID: "conn-1", Provider: "anthropic", RefreshToken: &secondToken}
	refresher := newBlockingRefresh(oauth.TokenResult{AccessToken: "access-token"}, nil)

	errs := make(chan error, 2)
	for _, conn := range []*store.Connection{firstConn, secondConn} {
		go func(conn *store.Connection) {
			_, err := manager.Refresh(context.Background(), conn, refresher.Refresh)
			errs <- err
		}(conn)
	}

	refresher.waitForCalls(t, 2)
	refresher.release()

	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("Refresh returned error: %v", err)
		}
	}
}

func TestRefreshManagerWrapsRefreshError(t *testing.T) {
	manager := NewRefreshManager()
	refreshToken := "refresh-token"
	conn := &store.Connection{ID: "conn-1", Provider: "anthropic", RefreshToken: &refreshToken}
	refreshErr := errors.New("provider unavailable")

	_, err := manager.Refresh(context.Background(), conn, func(ctx context.Context, conn *store.Connection) (oauth.TokenResult, error) {
		return oauth.TokenResult{}, refreshErr
	})
	if !errors.Is(err, refreshErr) {
		t.Fatalf("expected wrapped refresh error, got %v", err)
	}
}

func TestRefreshManagerRecoversFromPanicAndUnblocksWaiters(t *testing.T) {
	manager := NewRefreshManager()
	refreshToken := "refresh-token"
	conn := &store.Connection{ID: "conn-1", Provider: "anthropic", RefreshToken: &refreshToken}

	done := make(chan error, 1)
	go func() {
		_, err := manager.Refresh(context.Background(), conn, func(context.Context, *store.Connection) (oauth.TokenResult, error) {
			panic("boom")
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error from panicking refresh")
		}
	case <-time.After(time.Second):
		t.Fatal("waiter blocked after refresh panic")
	}

	// A subsequent refresh for the same key must not be stuck on a stale entry.
	want := oauth.TokenResult{AccessToken: "access-token"}
	got, err := manager.Refresh(context.Background(), conn, func(context.Context, *store.Connection) (oauth.TokenResult, error) {
		return want, nil
	})
	if err != nil {
		t.Fatalf("second refresh returned error: %v", err)
	}
	if got.AccessToken != want.AccessToken {
		t.Fatalf("access token = %q, want %q", got.AccessToken, want.AccessToken)
	}
}

func TestRefreshManagerConcurrentRefreshHammer(t *testing.T) {
	manager := NewRefreshManager()
	refreshToken := "refresh-token"
	conn := &store.Connection{ID: "conn-1", Provider: "anthropic", RefreshToken: &refreshToken}
	want := oauth.TokenResult{AccessToken: "access-token"}

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := manager.Refresh(context.Background(), conn, func(context.Context, *store.Connection) (oauth.TokenResult, error) {
				return want, nil
			})
			if err != nil {
				t.Errorf("Refresh returned error: %v", err)
				return
			}
			if got.AccessToken != want.AccessToken {
				t.Errorf("access token = %q, want %q", got.AccessToken, want.AccessToken)
			}
		}()
	}
	wg.Wait()
}

func assertTokenResult(t *testing.T, got, want oauth.TokenResult) {
	t.Helper()

	if got.AccessToken != want.AccessToken {
		t.Fatalf("access token = %q, want %q", got.AccessToken, want.AccessToken)
	}
	if got.RefreshToken != want.RefreshToken {
		t.Fatalf("refresh token = %q, want %q", got.RefreshToken, want.RefreshToken)
	}
}

type blockingRefresh struct {
	token       oauth.TokenResult
	err         error
	releaseOnce sync.Once
	releaseCh   chan struct{}

	mu    sync.Mutex
	count int
}

func newBlockingRefresh(token oauth.TokenResult, err error) *blockingRefresh {
	return &blockingRefresh{
		token:     token,
		err:       err,
		releaseCh: make(chan struct{}),
	}
}

func (r *blockingRefresh) Refresh(ctx context.Context, conn *store.Connection) (oauth.TokenResult, error) {
	r.mu.Lock()
	r.count++
	r.mu.Unlock()

	select {
	case <-r.releaseCh:
	case <-ctx.Done():
		return oauth.TokenResult{}, ctx.Err()
	}

	return r.token, r.err
}

func (r *blockingRefresh) waitForCalls(t *testing.T, want int) {
	t.Helper()

	deadline := time.After(time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()

	for {
		if r.calls() == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("refresh calls = %d, want %d", r.calls(), want)
		case <-tick.C:
		}
	}
}

func (r *blockingRefresh) waitForStableCalls(t *testing.T, want int) {
	t.Helper()

	deadline := time.After(time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()

	stable := 0
	for {
		if r.calls() == want {
			stable++
			if stable >= 10 {
				return
			}
		} else {
			stable = 0
		}
		select {
		case <-deadline:
			t.Fatalf("refresh calls = %d, want stable %d", r.calls(), want)
		case <-tick.C:
		}
	}
}

func (r *blockingRefresh) release() {
	r.releaseOnce.Do(func() {
		close(r.releaseCh)
	})
}

func (r *blockingRefresh) calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}
