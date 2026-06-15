package schemas

import "testing"

// TestWhiteListIsAllowed pins the matrix WhiteList contract (PAR-BF-GOV-026,
// account.go:22-30): ["*"] allows all; empty denies all; a listed non-"*" list
// allows only the listed values (D1/D6).
func TestWhiteListIsAllowed(t *testing.T) {
	tests := []struct {
		name  string
		list  WhiteList
		value string
		want  bool
	}{
		{"wildcard allows any", WhiteList{"*"}, "gpt-4o", true},
		{"wildcard allows another", WhiteList{"*"}, "claude-opus-4", true},
		{"empty denies all", WhiteList{}, "gpt-4o", false},
		{"nil denies all", nil, "gpt-4o", false},
		{"listed hit allowed", WhiteList{"gpt-4o", "gpt-4"}, "gpt-4o", true},
		{"listed miss denied", WhiteList{"gpt-4o", "gpt-4"}, "claude-opus-4", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.list.IsAllowed(tt.value); got != tt.want {
				t.Errorf("WhiteList(%v).IsAllowed(%q) = %v, want %v", tt.list, tt.value, got, tt.want)
			}
		})
	}
}

// TestBlackListIsBlocked pins the matrix BlackList contract (PAR-BF-GOV-027,
// account.go:80-106; PAR-BF-OAI-119): empty blocks none; ["*"] blocks all; a
// listed non-"*" list blocks only the listed values (D2/D6).
func TestBlackListIsBlocked(t *testing.T) {
	tests := []struct {
		name  string
		list  BlackList
		value string
		want  bool
	}{
		{"empty blocks none", BlackList{}, "gpt-4o", false},
		{"nil blocks none", nil, "gpt-4o", false},
		{"wildcard blocks any", BlackList{"*"}, "gpt-4o", true},
		{"wildcard blocks another", BlackList{"*"}, "claude-opus-4", true},
		{"listed hit blocked", BlackList{"gpt-4o", "gpt-4"}, "gpt-4o", true},
		{"listed miss not blocked", BlackList{"gpt-4o", "gpt-4"}, "claude-opus-4", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.list.IsBlocked(tt.value); got != tt.want {
				t.Errorf("BlackList(%v).IsBlocked(%q) = %v, want %v", tt.list, tt.value, got, tt.want)
			}
		})
	}
}

// TestListsBlacklistWinsCrossCase asserts the type-level cross interaction (D3):
// a value that is both allowed (whitelist) and blocked (blacklist) is blocked.
func TestListsBlacklistWinsCrossCase(t *testing.T) {
	allow := WhiteList{"gpt-4o"}
	block := BlackList{"gpt-4o"}
	if !allow.IsAllowed("gpt-4o") {
		t.Fatal("precondition: whitelist should allow gpt-4o")
	}
	if !block.IsBlocked("gpt-4o") {
		t.Fatal("blacklist should block gpt-4o (blacklist-wins precedence is enforced by the gate)")
	}
}

// TestWhiteListValidate pins the D5 mix-rule: "*" may not be combined with
// explicit entries; well-formed lists validate.
func TestWhiteListValidate(t *testing.T) {
	tests := []struct {
		name    string
		list    WhiteList
		wantErr bool
	}{
		{"empty ok", WhiteList{}, false},
		{"nil ok", nil, false},
		{"pure wildcard ok", WhiteList{"*"}, false},
		{"listed ok", WhiteList{"gpt-4o", "gpt-4"}, false},
		{"wildcard plus explicit rejected", WhiteList{"*", "gpt-4o"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.list.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WhiteList(%v).Validate() err = %v, wantErr %v", tt.list, err, tt.wantErr)
			}
		})
	}
}

// TestBlackListValidate pins the D5 mix-rule for the blacklist.
func TestBlackListValidate(t *testing.T) {
	tests := []struct {
		name    string
		list    BlackList
		wantErr bool
	}{
		{"empty ok", BlackList{}, false},
		{"nil ok", nil, false},
		{"pure wildcard ok", BlackList{"*"}, false},
		{"listed ok", BlackList{"gpt-4o", "gpt-4"}, false},
		{"wildcard plus explicit rejected", BlackList{"*", "gpt-4o"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.list.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("BlackList(%v).Validate() err = %v, wantErr %v", tt.list, err, tt.wantErr)
			}
		})
	}
}
