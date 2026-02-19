package main

import (
	"github.com/MyAgentHubs/aimemo/internal/cli"
)

var version = "dev"

func main() {
	cli.Execute(version)
}
