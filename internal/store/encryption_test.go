package store

import "testing"

func TestSetEncKeyEmptyClearsKey(t *testing.T) {
	s := &Store{}
	s.SetEncKey("secret")
	if s.encKey == nil {
		t.Fatal("expected encKey to be set")
	}
	s.SetEncKey("")
	if s.encKey != nil {
		t.Fatal("expected encKey to be cleared")
	}
}
