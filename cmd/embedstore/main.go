package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gamzabox/embedstore"
)

func main() {
	if len(os.Args) < 2 {
		die("usage: embedstore <validate|build|search|inspect|verify>")
	}
	switch os.Args[1] {
	case "validate":
		validate(os.Args[2:])
	case "build":
		build(os.Args[2:])
	case "search":
		search(os.Args[2:])
	case "inspect":
		inspect(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	default:
		die("unknown command")
	}
}
func verify(args []string) {
	f := flag.NewFlagSet("verify", flag.ExitOnError)
	path := f.String("file", "", "embed file")
	f.Parse(args)
	require(*path, "--file")
	m, err := embedstore.VerifyFile(*path)
	if err != nil {
		die(err.Error())
	}
	fmt.Printf("Verification successful\nItems: %d\n", m.ItemCount)
}
func require(v, n string) {
	if v == "" {
		die(n + " is required")
	}
}
func die(s string) { fmt.Fprintln(os.Stderr, "error:", s); os.Exit(1) }
