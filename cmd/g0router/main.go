package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bloodf/g0router"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/server"
	"github.com/bloodf/g0router/internal/store"
)

var (
	version   = "0.2.0-dev"
	buildDate = ""
)

const (
	defaultListen = ":20128"

	// Default admin credentials seeded on first run only. Change the
	// password via the dashboard; no env vars are involved.
	defaultAdminUser     = "admin"
	defaultAdminPassword = "123456"
)

func parseAllowedOrigins() []string {
	raw := os.Getenv("G0ROUTER_ALLOWED_ORIGINS")
	if raw == "" {
		return nil
	}
	var out []string
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			out = append(out, o)
		}
	}
	return out
}

func resolveDataDir() string {
	dataDir := os.Getenv("G0ROUTER_DATA")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("resolve home dir: %v", err)
		}
		dataDir = filepath.Join(home, ".g0router")
	}
	return dataDir
}

func openStore(dataDir string) (*store.Store, error) {
	secret, err := store.LoadOrCreateSecret(dataDir)
	if err != nil {
		return nil, fmt.Errorf("load encryption secret: %w", err)
	}
	st, err := store.Open(filepath.Join(dataDir, "g0router.db"), secret)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	return st, nil
}

func resetPassword(dataDir string) error {
	st, err := openStore(dataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	user, err := st.FirstUser()
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if err := st.SetUserPasswordHash(user.Username, ""); err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	fmt.Println("Password reset to default.")
	return nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "reset-password" {
		dataDir := resolveDataDir()
		if err := resetPassword(dataDir); err != nil {
			log.Fatalf("reset-password: %v", err)
		}
		return
	}

	listenAddr := os.Getenv("G0ROUTER_LISTEN")
	if listenAddr == "" {
		listenAddr = defaultListen
	}

	dataDir := resolveDataDir()
	st, err := openStore(dataDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	sessions := auth.NewSessions(st, 7*24*time.Hour)
	created, err := sessions.SeedAdmin(defaultAdminUser, defaultAdminPassword)
	if err != nil {
		log.Fatalf("seed admin user: %v", err)
	}
	if created {
		log.Printf("seeded default admin user %q — change the password via the dashboard", defaultAdminUser)
	}

	uiFS, err := g0router.UI()
	if err != nil {
		log.Fatalf("open embedded ui: %v", err)
	}

	allowedOrigins := parseAllowedOrigins()
	srv := server.NewWithShutdown(uiFS, st, allowedOrigins)

	versionLine := version
	if buildDate != "" {
		versionLine = fmt.Sprintf("%s (built %s)", version, buildDate)
	}
	log.Printf("g0router %s listening on %s", versionLine, listenAddr)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.ListenAndServe(listenAddr)
	}()

	select {
	case sig := <-shutdown:
		log.Printf("received %s, shutting down", sig)
		if err := srv.Close(); err != nil {
			log.Fatalf("shutdown: %v", err)
		}
		if err := <-serveErr; err != nil {
			log.Fatalf("listen %s: %v", listenAddr, err)
		}
		return
	case err := <-serveErr:
		log.Fatalf("listen %s: %v", listenAddr, err)
	}
}
