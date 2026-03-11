package main

import "github.com/RazBrry/safe-ify/internal/cli"

// version is set at build time via ldflags.
var version = "dev"

func main() {
	cli.Execute(version)
}
