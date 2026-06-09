package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bloodf/g0router"
	"github.com/bloodf/g0router/internal/server"
)

var (
	version   = "0.2.0-dev"
	buildDate = ""
)

const defaultListen = ":20128"

func main() {
	listenAddr := os.Getenv("G0ROUTER_LISTEN")
	if listenAddr == "" {
		listenAddr = defaultListen
	}

	uiFS, err := g0router.UI()
	if err != nil {
		log.Fatalf("open embedded ui: %v", err)
	}

	srv := server.New(uiFS)

	versionLine := version
	if buildDate != "" {
		versionLine = fmt.Sprintf("%s (built %s)", version, buildDate)
	}
	log.Printf("g0router %s listening on %s", versionLine, listenAddr)

	if err := srv.ListenAndServe(listenAddr); err != nil {
		log.Fatalf("listen %s: %v", listenAddr, err)
	}
}
