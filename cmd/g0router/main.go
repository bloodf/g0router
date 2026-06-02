package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "0.1.0-dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	fmt.Fprintf(os.Stderr, "g0router %s\n", version)
	fmt.Fprintln(os.Stderr, "Use 'g0router serve' to start the server (not yet implemented)")
	os.Exit(1)
}
