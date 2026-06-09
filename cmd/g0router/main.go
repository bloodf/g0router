package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

func main() {
	listenAddr := os.Getenv("G0ROUTER_LISTEN")
	if listenAddr == "" {
		listenAddr = defaultListen
	}

	dataDir := os.Getenv("G0ROUTER_DATA")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("resolve home dir: %v", err)
		}
		dataDir = filepath.Join(home, ".g0router")
	}

	secret, err := store.LoadOrCreateSecret(dataDir)
	if err != nil {
		log.Fatalf("load encryption secret: %v", err)
	}
	st, err := store.Open(filepath.Join(dataDir, "g0router.db"), secret)
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

	srv := server.New(uiFS, st)

	versionLine := version
	if buildDate != "" {
		versionLine = fmt.Sprintf("%s (built %s)", version, buildDate)
	}
	log.Printf("g0router %s listening on %s", versionLine, listenAddr)

	if err := srv.ListenAndServe(listenAddr); err != nil {
		log.Fatalf("listen %s: %v", listenAddr, err)
	}
}
