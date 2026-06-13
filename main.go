// Command pac is one front door for pacman (official + aur-mirror) and flatpak.
package main

import (
	"os"

	"starnix.net/pac/internal/cli"
	"starnix.net/pac/internal/run"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], run.Real{}, os.Stdout, os.Stderr))
}
