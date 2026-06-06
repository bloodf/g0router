package store

import (
	"testing"
)

func TestMCPToolGroupClosedStoreErrorPaths(t *testing.T) {
	s := closedStore(t)

	checks := []struct {
		name string
		fn   func() error
	}{
		{"ListMCPToolGroups", func() error { _, err := s.ListMCPToolGroups(); return err }},
		{"GetMCPToolGroup", func() error { _, err := s.GetMCPToolGroup(1); return err }},
		{"GetMCPToolGroupByName", func() error { _, err := s.GetMCPToolGroupByName("x"); return err }},
		{"CreateMCPToolGroup", func() error { _, err := s.CreateMCPToolGroup("x", nil, true); return err }},
		{"UpdateMCPToolGroup", func() error { return s.UpdateMCPToolGroup(1, "x", nil, true) }},
		{"DeleteMCPToolGroup", func() error { return s.DeleteMCPToolGroup(1) }},
	}

	for _, c := range checks {
		if err := c.fn(); err == nil {
			t.Errorf("%s on closed DB: want error", c.name)
		}
	}
}
