package store

import "testing"

// TestConnectionProxyPoolIDRoundtrip proves the additive connections.proxy_pool_id
// linkage round-trips through create/get/list and is reflected by
// CountConnectionsUsingProxyPool (the bound-connection delete guard).
func TestConnectionProxyPoolIDRoundtrip(t *testing.T) {
	st := newTestStore(t)

	pool, err := st.CreateProxyPool(&ProxyPool{Name: "p", Host: "h.example.com"})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	if err := st.CreateProvider(&ProviderRecord{Name: "prov", Type: "openai"}); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	provs, err := st.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}

	conn := &Connection{ProviderID: provs[0].ID, Name: "c", Kind: "api_key", Secret: "k", ProxyPoolID: pool.ID}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	got, err := st.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.ProxyPoolID != pool.ID {
		t.Fatalf("proxy_pool_id did not round-trip: got %q want %q", got.ProxyPoolID, pool.ID)
	}

	list, err := st.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(list) != 1 || list[0].ProxyPoolID != pool.ID {
		t.Fatalf("list did not carry proxy_pool_id: %+v", list)
	}

	n, err := st.CountConnectionsUsingProxyPool(pool.ID)
	if err != nil {
		t.Fatalf("CountConnectionsUsingProxyPool: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 bound connection, got %d", n)
	}
}
