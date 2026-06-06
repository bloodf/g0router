package store

import (
	"errors"
	"testing"
)

func TestProxyPoolCreateListGetRoundTrip(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := ProxyPool{
		Name:     "us-east-1",
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     8080,
		Username: "user1",
		Password: "secret123",
		IsActive: true,
	}

	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	if created.ID == "" {
		t.Fatal("ID should be set after create")
	}
	if created.CreatedAt == "" {
		t.Fatal("CreatedAt should be set after create")
	}
	if created.Password != "" {
		t.Fatal("Password should be cleared after create")
	}

	list, err := s.ListProxyPools()
	if err != nil {
		t.Fatalf("ListProxyPools: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	if list[0].Password != "secret123" {
		t.Fatalf("password decrypted = %q, want secret123", list[0].Password)
	}

	got, err := s.GetProxyPool(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if got.Name != "us-east-1" {
		t.Fatalf("name = %q, want us-east-1", got.Name)
	}
	if got.Password != "secret123" {
		t.Fatalf("password decrypted = %q, want secret123", got.Password)
	}
}

func TestProxyPoolPasswordEncryptedInDB(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := ProxyPool{
		Name:     "us-east-1",
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     8080,
		Password: "secret123",
		IsActive: true,
	}

	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	var passwordEnc string
	err = s.db.QueryRow("SELECT password_enc FROM proxy_pools WHERE id = ?", created.ID).Scan(&passwordEnc)
	if err != nil {
		t.Fatalf("query db: %v", err)
	}
	if passwordEnc == "" {
		t.Fatal("password_enc should not be empty")
	}
	if passwordEnc == "secret123" {
		t.Fatal("password_enc should be encrypted, not plaintext")
	}

	got, err := s.GetProxyPool(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if got.Password != "secret123" {
		t.Fatalf("password decrypted = %q, want secret123", got.Password)
	}
}

func TestProxyPoolUpdate(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := ProxyPool{
		Name:     "us-east-1",
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     8080,
		Password: "secret123",
		IsActive: true,
	}

	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	updated := ProxyPool{
		Name:     "us-east-2",
		Protocol: "https",
		Host:     "proxy2.example.com",
		Port:     9090,
		Username: "user2",
		Password: "newsecret",
		IsActive: false,
	}

	if err := s.UpdateProxyPool(created.ID, updated); err != nil {
		t.Fatalf("UpdateProxyPool: %v", err)
	}

	got, err := s.GetProxyPool(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if got.Name != "us-east-2" {
		t.Fatalf("name = %q, want us-east-2", got.Name)
	}
	if got.Protocol != "https" {
		t.Fatalf("protocol = %q, want https", got.Protocol)
	}
	if got.Host != "proxy2.example.com" {
		t.Fatalf("host = %q, want proxy2.example.com", got.Host)
	}
	if got.Port != 9090 {
		t.Fatalf("port = %d, want 9090", got.Port)
	}
	if got.Username != "user2" {
		t.Fatalf("username = %q, want user2", got.Username)
	}
	if got.Password != "newsecret" {
		t.Fatalf("password = %q, want newsecret", got.Password)
	}
	if got.IsActive {
		t.Fatal("should be inactive")
	}
}

func TestProxyPoolUpdateWithoutPasswordPreservesExisting(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := ProxyPool{
		Name:     "us-east-1",
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     8080,
		Password: "secret123",
		IsActive: true,
	}

	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	updated := ProxyPool{
		Name:     "renamed",
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     8080,
		IsActive: true,
	}

	if err := s.UpdateProxyPool(created.ID, updated); err != nil {
		t.Fatalf("UpdateProxyPool: %v", err)
	}

	got, err := s.GetProxyPool(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if got.Password != "secret123" {
		t.Fatalf("password = %q, want secret123", got.Password)
	}
}

func TestProxyPoolDelete(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := ProxyPool{
		Name:     "us-east-1",
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     8080,
		IsActive: true,
	}

	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	if err := s.DeleteProxyPool(created.ID); err != nil {
		t.Fatalf("DeleteProxyPool: %v", err)
	}

	_, err = s.GetProxyPool(created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProxyPoolDeleteNotFound(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	err := s.DeleteProxyPool("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProxyPoolGetNotFound(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	_, err := s.GetProxyPool("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProxyPoolTestPlaceholder(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ok, latency, err := s.TestProxyPool("any-id")
	if err != nil {
		t.Fatalf("TestProxyPool: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true")
	}
	if latency != 0 {
		t.Fatalf("latency = %d, want 0", latency)
	}
}
