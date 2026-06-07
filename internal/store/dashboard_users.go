package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type DashboardUser struct {
	ID           string
	Username     string
	PasswordHash string
	DisplayName  string
	Role         string
	CreatedAt    string
}

// ErrDashboardUserExists is returned when a username already exists.
var ErrDashboardUserExists = errors.New("dashboard user already exists")

// ErrInvalidDashboardUserPassword is returned when a password fails validation.
var ErrInvalidDashboardUserPassword = errors.New("invalid dashboard user password: must be at least 8 characters and not whitespace-only")

// ErrInvalidDashboardUserRole is returned when a role is not "admin" or "user".
var ErrInvalidDashboardUserRole = errors.New("invalid dashboard user role: must be admin or user")

func validateDashboardUserPassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return ErrInvalidDashboardUserPassword
	}
	if len(password) < 8 {
		return ErrInvalidDashboardUserPassword
	}
	return nil
}

func validateDashboardUserRole(role string) error {
	if role == "" {
		return ErrInvalidDashboardUserRole
	}
	if role != "admin" && role != "user" {
		return ErrInvalidDashboardUserRole
	}
	return nil
}

// isUniqueConstraintError checks whether err is a SQLite unique-constraint
// violation. It relies on the error string from modernc.org/sqlite; if the
// driver changes its message format this helper will need updating.
func isUniqueConstraintError(err error, table string) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: "+table)
}

// SeedDefaultAdminUser creates the built-in admin account used for first-time
// setup. It bypasses password-length validation so that the initial password
// can be short (the operator is expected to change it immediately).
func (s *Store) SeedDefaultAdminUser(username, password, displayName, role string) (*DashboardUser, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var user DashboardUser
	err = s.db.QueryRow(
		`INSERT INTO dashboard_users (username, password_hash, display_name, role)
		VALUES (?, ?, ?, ?)
		RETURNING id, username, password_hash, display_name, role, created_at`,
		username,
		string(hash),
		displayName,
		role,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName, &user.Role, &user.CreatedAt)
	if err != nil {
		if isUniqueConstraintError(err, "dashboard_users") {
			return nil, fmt.Errorf("create dashboard user: %w", ErrDashboardUserExists)
		}
		return nil, fmt.Errorf("create dashboard user: %w", err)
	}

	return &user, nil
}

func (s *Store) CreateDashboardUser(username, password, displayName, role string) (*DashboardUser, error) {
	if err := validateDashboardUserPassword(password); err != nil {
		return nil, err
	}

	if role == "" {
		role = "user"
	}
	if err := validateDashboardUserRole(role); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	var user DashboardUser
	err = s.db.QueryRow(
		`INSERT INTO dashboard_users (username, password_hash, display_name, role)
		VALUES (?, ?, ?, ?)
		RETURNING id, username, password_hash, display_name, role, created_at`,
		username,
		string(hash),
		displayName,
		role,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName, &user.Role, &user.CreatedAt)
	if err != nil {
		if isUniqueConstraintError(err, "dashboard_users") {
			return nil, fmt.Errorf("create dashboard user: %w", ErrDashboardUserExists)
		}
		return nil, fmt.Errorf("create dashboard user: %w", err)
	}

	return &user, nil
}

func scanDashboardUser(scan func(dest ...any) error) (*DashboardUser, error) {
	var user DashboardUser
	if err := scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName, &user.Role, &user.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetDashboardUser(id string) (*DashboardUser, error) {
	row := s.db.QueryRow(
		`SELECT id, username, password_hash, display_name, role, created_at FROM dashboard_users WHERE id = ?`,
		id,
	)
	user, err := scanDashboardUser(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get dashboard user: user %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get dashboard user: %w", err)
	}
	return user, nil
}

func (s *Store) GetDashboardUserByUsername(username string) (*DashboardUser, error) {
	row := s.db.QueryRow(
		`SELECT id, username, password_hash, display_name, role, created_at FROM dashboard_users WHERE username = ?`,
		username,
	)
	user, err := scanDashboardUser(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get dashboard user by username: user %q not found", username)
	}
	if err != nil {
		return nil, fmt.Errorf("get dashboard user by username: %w", err)
	}
	return user, nil
}

func (s *Store) ListDashboardUsers() ([]DashboardUser, error) {
	rows, err := s.db.Query(
		`SELECT id, username, password_hash, display_name, role, created_at
		FROM dashboard_users
		ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("query dashboard users: %w", err)
	}
	defer rows.Close()

	var users []DashboardUser
	for rows.Next() {
		user, err := scanDashboardUser(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan dashboard user: %w", err)
		}
		users = append(users, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dashboard users: %w", err)
	}

	return users, nil
}

func (s *Store) UpdateDashboardUser(id string, username, displayName, role *string) (*DashboardUser, error) {
	if role != nil {
		if err := validateDashboardUserRole(*role); err != nil {
			return nil, err
		}
	}

	// Build partial update dynamically
	setClauses := []string{}
	args := []any{}
	if username != nil {
		setClauses = append(setClauses, "username = ?")
		args = append(args, *username)
	}
	if displayName != nil {
		setClauses = append(setClauses, "display_name = ?")
		args = append(args, *displayName)
	}
	if role != nil {
		setClauses = append(setClauses, "role = ?")
		args = append(args, *role)
	}

	if len(setClauses) == 0 {
		// Nothing to update; return current user
		return s.GetDashboardUser(id)
	}

	args = append(args, id)
	query := fmt.Sprintf(
		"UPDATE dashboard_users SET %s WHERE id = ? RETURNING id, username, password_hash, display_name, role, created_at",
		strings.Join(setClauses, ", "),
	)

	var user DashboardUser
	err := s.db.QueryRow(query, args...).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.DisplayName, &user.Role, &user.CreatedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err, "dashboard_users") {
			return nil, fmt.Errorf("update dashboard user: %w", ErrDashboardUserExists)
		}
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("update dashboard user: user %q not found", id)
		}
		return nil, fmt.Errorf("update dashboard user: %w", err)
	}

	return &user, nil
}

func (s *Store) DeleteDashboardUser(id string) error {
	res, err := s.db.Exec("DELETE FROM dashboard_users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete dashboard user: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete dashboard user rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("delete dashboard user: user %q not found", id)
	}
	return nil
}

func (s *Store) VerifyDashboardUserPassword(user *DashboardUser, password string) bool {
	if user == nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

func (s *Store) UpdateDashboardUserPassword(id string, newPassword string) error {
	if err := validateDashboardUserPassword(newPassword); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	res, err := s.db.Exec(
		"UPDATE dashboard_users SET password_hash = ? WHERE id = ?",
		string(hash),
		id,
	)
	if err != nil {
		return fmt.Errorf("update dashboard user password: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update dashboard user password rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("update dashboard user password: user %q not found", id)
	}
	return nil
}
