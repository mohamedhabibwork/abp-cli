package main

import (
	"github.com/mohamedhabibwork/abp-cli/internal/abpcrud"
	"os"
)

func main() {
	os.Exit(abpcrud.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
