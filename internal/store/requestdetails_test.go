package store

import (
	"encoding/json"
	"testing"
)

func TestSaveRequestDetailsAndQuery(t *testing.T) {
	st := newTestStore(t)

	items := []*RequestDetailRow{
		{ID: "id-1", Timestamp: "2026-06-12T10:00:00Z", Provider: "openai", Model: "gpt-4o", ConnectionID: "conn-a", Status: "ok", Data: json.RawMessage(`{"id":"id-1"}`)},
		{ID: "id-2", Timestamp: "2026-06-12T10:00:01Z", Provider: "anthropic", Model: "claude-3", ConnectionID: "conn-b", Status: "error", Data: json.RawMessage(`{"id":"id-2"}`)},
	}
	if err := st.SaveRequestDetails(items, 200); err != nil {
		t.Fatalf("SaveRequestDetails: %v", err)
	}

	rows, _, err := st.QueryRequestDetails(RequestDetailsFilter{})
	if err != nil {
		t.Fatalf("QueryRequestDetails: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
}

func TestSaveRequestDetailsUpsertsAndRetention(t *testing.T) {
	st := newTestStore(t)

	for i := 0; i < 5; i++ {
		items := []*RequestDetailRow{
			{ID: "id-" + string(rune('0'+i)), Timestamp: "2026-06-12T10:00:0" + string(rune('0'+i)) + "Z", Provider: "openai", Model: "gpt-4o", Data: json.RawMessage(`{"id":"id"}`)},
		}
		if err := st.SaveRequestDetails(items, 3); err != nil {
			t.Fatalf("SaveRequestDetails %d: %v", i, err)
		}
	}

	rows, _, err := st.QueryRequestDetails(RequestDetailsFilter{})
	if err != nil {
		t.Fatalf("QueryRequestDetails: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}

	got, err := st.GetRequestDetailByID("id-4")
	if err != nil {
		t.Fatalf("GetRequestDetailByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetRequestDetailByID returned nil")
	}
	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if parsed["id"] != "id" {
		t.Errorf("data.id = %v, want id", parsed["id"])
	}
}

func TestRequestDetailsQuery(t *testing.T) {
	st := newTestStore(t)

	seed := []struct {
		id, ts, provider, model, conn, status string
	}{
		{"d1", "2026-06-12T10:00:00Z", "openai", "gpt-4o", "conn-1", "ok"},
		{"d2", "2026-06-12T10:00:01Z", "openai", "gpt-4o-mini", "conn-1", "error"},
		{"d3", "2026-06-12T10:00:02Z", "anthropic", "claude-3", "conn-2", "ok"},
		{"d4", "2026-06-12T10:00:03Z", "anthropic", "claude-3", "conn-2", "ok"},
		{"d5", "2026-06-12T10:00:04Z", "openai", "gpt-4o", "conn-1", "error"},
		{"d6", "2026-06-12T10:00:05Z", "openai", "gpt-4o", "conn-1", "ok"},
	}

	var items []*RequestDetailRow
	for _, s := range seed {
		data, _ := json.Marshal(map[string]any{"id": s.id})
		items = append(items, &RequestDetailRow{
			ID:           s.id,
			Timestamp:    s.ts,
			Provider:     s.provider,
			Model:        s.model,
			ConnectionID: s.conn,
			Status:       s.status,
			Data:         data,
		})
	}
	if err := st.SaveRequestDetails(items, 200); err != nil {
		t.Fatalf("SaveRequestDetails: %v", err)
	}

	t.Run("filter by provider", func(t *testing.T) {
		rows, pg, err := st.QueryRequestDetails(RequestDetailsFilter{Provider: "openai"})
		if err != nil {
			t.Fatalf("QueryRequestDetails: %v", err)
		}
		if len(rows) != 4 {
			t.Errorf("len(rows) = %d, want 4", len(rows))
		}
		if pg.TotalItems != 4 {
			t.Errorf("TotalItems = %d, want 4", pg.TotalItems)
		}
	})

	t.Run("filter by provider and status", func(t *testing.T) {
		rows, _, err := st.QueryRequestDetails(RequestDetailsFilter{Provider: "openai", Status: "error"})
		if err != nil {
			t.Fatalf("QueryRequestDetails: %v", err)
		}
		if len(rows) != 2 {
			t.Errorf("len(rows) = %d, want 2", len(rows))
		}
	})

	t.Run("pagination page 2", func(t *testing.T) {
		rows, pg, err := st.QueryRequestDetails(RequestDetailsFilter{Page: 2, PageSize: 2})
		if err != nil {
			t.Fatalf("QueryRequestDetails: %v", err)
		}
		if len(rows) != 2 {
			t.Errorf("len(rows) = %d, want 2", len(rows))
		}
		if pg.Page != 2 {
			t.Errorf("Page = %d, want 2", pg.Page)
		}
		if pg.TotalItems != 6 {
			t.Errorf("TotalItems = %d, want 6", pg.TotalItems)
		}
		if pg.TotalPages != 3 {
			t.Errorf("TotalPages = %d, want 3", pg.TotalPages)
		}
		if !pg.HasNext {
			t.Error("HasNext = false, want true")
		}
		if !pg.HasPrev {
			t.Error("HasPrev = false, want true")
		}

		var first map[string]any
		if err := json.Unmarshal(rows[0], &first); err != nil {
			t.Fatalf("unmarshal first row: %v", err)
		}
		if first["id"] != "d4" {
			t.Errorf("first row id = %v, want d4", first["id"])
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		rows, _, err := st.QueryRequestDetails(RequestDetailsFilter{StartDate: "2026-06-12T10:00:01Z", EndDate: "2026-06-12T10:00:03Z"})
		if err != nil {
			t.Fatalf("QueryRequestDetails: %v", err)
		}
		if len(rows) != 3 {
			t.Errorf("len(rows) = %d, want 3", len(rows))
		}
	})
}

func TestRequestDetailByID(t *testing.T) {
	st := newTestStore(t)

	if err := st.SaveRequestDetails([]*RequestDetailRow{
		{ID: "found", Timestamp: "2026-06-12T10:00:00Z", Data: json.RawMessage(`{"x":1}`)},
	}, 200); err != nil {
		t.Fatalf("SaveRequestDetails: %v", err)
	}

	got, err := st.GetRequestDetailByID("found")
	if err != nil {
		t.Fatalf("GetRequestDetailByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected row, got nil")
	}
	var parsed map[string]any
	if err := json.Unmarshal(got, &parsed); err != nil || parsed["x"] != float64(1) {
		t.Errorf("data = %s, want {\"x\":1}", string(got))
	}

	missing, err := st.GetRequestDetailByID("missing")
	if err != nil {
		t.Fatalf("GetRequestDetailByID missing: %v", err)
	}
	if missing != nil {
		t.Errorf("missing = %s, want nil", string(missing))
	}
}

