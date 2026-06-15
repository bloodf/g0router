package server

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// enableVKMandatoryFlag flips the seeded vk_mandatory flag ON (bf-gov-4, D3) so
// the wiring test proves the per-request predicate reads the live flag store.
func enableVKMandatoryFlag(t *testing.T, st *store.Store) {
	t.Helper()
	if _, err := st.DB().Exec(
		"UPDATE feature_flags SET enabled = 1 WHERE key = 'vk_mandatory'",
	); err != nil {
		t.Fatalf("enable vk_mandatory flag: %v", err)
	}
}

// TestVKMandatoryWiringRejectsAbsentVKWhenFlagOn proves the predicate is wired
// LIVE end-to-end (bf-gov-4, D3): with vk_mandatory ON, a /v1/chat/completions
// request bearing NO x-g0-vk is rejected 401 "virtual key required", and with
// the flag OFF (seeded default) the same request is admitted (reaches the
// provider, 200). The single AllowVK gate seam enforces this with no
// per-call-site mandatory logic — only the if-vkHeader guard is removed so
// AllowVK("") is reachable (Option A authorization).
func TestVKMandatoryWiringRejectsAbsentVKWhenFlagOn(t *testing.T) {
	t.Run("flag ON + no x-g0-vk rejected 401", func(t *testing.T) {
		st := newTestStore(t)
		var hits int64
		defer countingStubCatalog(t, &hits)()

		sessions := auth.NewSessions(st, time.Hour)
		if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
			t.Fatalf("SeedAdmin: %v", err)
		}
		seedSmokeProvider(t, st)
		enableVKMandatoryFlag(t, st)

		rec, err := st.CreateAPIKey("vkmand-on")
		if err != nil {
			t.Fatalf("CreateAPIKey: %v", err)
		}
		srv := New(testUIFS(), st, nil)

		ctx := chatRequest(t, srv, rec.Key) // sends Authorization only, no x-g0-vk
		if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
			t.Fatalf("flag ON + no VK status = %d, want 401: %s", ctx.Response.StatusCode(), ctx.Response.Body())
		}
		if got := atomic.LoadInt64(&hits); got != 0 {
			t.Fatalf("provider hits = %d, want 0 (request rejected at the gate before upstream)", got)
		}
	})

	t.Run("flag OFF + no x-g0-vk admitted", func(t *testing.T) {
		st := newTestStore(t)
		var hits int64
		defer countingStubCatalog(t, &hits)()

		sessions := auth.NewSessions(st, time.Hour)
		if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
			t.Fatalf("SeedAdmin: %v", err)
		}
		seedSmokeProvider(t, st)
		// vk_mandatory left OFF (seeded disabled by the migration).

		rec, err := st.CreateAPIKey("vkmand-off")
		if err != nil {
			t.Fatalf("CreateAPIKey: %v", err)
		}
		srv := New(testUIFS(), st, nil)

		ctx := chatRequest(t, srv, rec.Key)
		if ctx.Response.StatusCode() != fasthttp.StatusOK {
			t.Fatalf("flag OFF + no VK status = %d, want 200 (admitted): %s", ctx.Response.StatusCode(), ctx.Response.Body())
		}
		if got := atomic.LoadInt64(&hits); got != 1 {
			t.Fatalf("provider hits = %d, want 1 (flag off => request reaches upstream)", got)
		}
	})
}
