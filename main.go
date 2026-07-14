// Command bwenv provides app-scoped environments backed by Bitwarden Secrets Manager.
package main

import (
	"os"

	"bwenv/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
