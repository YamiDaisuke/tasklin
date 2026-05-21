package main

import "github.com/frankcruz/tasklin/cmd"

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	cmd.Execute(version, commit, buildDate)
}
