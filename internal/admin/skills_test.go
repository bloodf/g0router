package admin

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestListSkills(t *testing.T) {
	env := newTestEnv(t)

	status, envl := call(t, env.handlers.ListSkills, "GET", "/api/skills", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list skills status = %d", status)
	}
	skills := dataField[[]map[string]any](t, envl)
	if len(skills) < 2 {
		t.Fatalf("skills len = %d, want >= 2", len(skills))
	}

	// Flat shape {name,category,description,url} (§1.2).
	first := skills[0]
	for _, key := range []string{"name", "category", "description", "url"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("skill missing key %q: %v", key, first)
		}
	}

	// At least one category is shared across >= 2 skills (grouping surface).
	byCategory := map[string]int{}
	for _, s := range skills {
		cat, _ := s["category"].(string)
		byCategory[cat]++
	}
	grouped := false
	for _, n := range byCategory {
		if n >= 2 {
			grouped = true
		}
	}
	if !grouped {
		t.Fatalf("no category groups >= 2 skills: %v", byCategory)
	}
}
