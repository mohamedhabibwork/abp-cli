package main

import (
	"os"

	"github.com/mohamedhabibwork/abp-cli/internal/abpcrud"
)

func main() {
	os.Exit(abpcrud.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
