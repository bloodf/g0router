package schemas

import "fmt"

// wildcard is the literal match-all sentinel for both list types (D6,
// matrix PAR-BF-GOV-026/027). No glob/prefix matching is implemented.
const wildcard = "*"

// WhiteList is an allow-list of model identifiers (PAR-BF-GOV-026, account.go:22-30).
// Its IsAllowed encodes the matrix contract: ["*"] allows all, empty denies all,
// and a listed non-wildcard list allows only the listed values.
type WhiteList []string

// IsAllowed reports whether value is permitted by the whitelist (D1):
//   - contains "*" -> true (allow all),
//   - empty -> false (deny all, matrix 026),
//   - else -> exact-string membership.
func (w WhiteList) IsAllowed(value string) bool {
	for _, m := range w {
		if m == wildcard {
			return true
		}
	}
	if len(w) == 0 {
		return false
	}
	for _, m := range w {
		if m == value {
			return true
		}
	}
	return false
}

// Validate enforces the minimal documented invariant (D5): the wildcard "*"
// may not be combined with explicit entries (it already covers all).
func (w WhiteList) Validate() error {
	return validateList("whitelist", w)
}

// BlackList is a block-list of model identifiers (PAR-BF-GOV-027, account.go:80-106;
// PAR-BF-OAI-119). Its IsBlocked encodes the matrix contract: empty blocks none,
// ["*"] blocks all, and a listed non-wildcard list blocks only the listed values.
type BlackList []string

// IsBlocked reports whether value is blocked by the blacklist (D2). Empty is
// checked first so an unconfigured blacklist never blocks (matrix 027):
//   - empty -> false (block none),
//   - contains "*" -> true (block all),
//   - else -> exact-string membership.
func (b BlackList) IsBlocked(value string) bool {
	if len(b) == 0 {
		return false
	}
	for _, m := range b {
		if m == wildcard {
			return true
		}
	}
	for _, m := range b {
		if m == value {
			return true
		}
	}
	return false
}

// Validate enforces the minimal documented invariant (D5): the wildcard "*"
// may not be combined with explicit entries.
func (b BlackList) Validate() error {
	return validateList("blacklist", b)
}

// validateList rejects a list that mixes the wildcard "*" with explicit entries
// (ambiguous/redundant). A pure wildcard, an empty list, and a purely-explicit
// list are all well-formed.
func validateList(kind string, list []string) error {
	if len(list) <= 1 {
		return nil
	}
	for _, m := range list {
		if m == wildcard {
			return fmt.Errorf("%s: '*' cannot be combined with explicit models", kind)
		}
	}
	return nil
}
