package main

import (
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		os.Exit(1)
	}
}
