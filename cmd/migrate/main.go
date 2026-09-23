package main

import (
	"os"
)

func main() {
	// run already logged a classified terminal record. The error text is not
	// printed because it can carry the DSN or failing SQL.
	if err := run(os.Args[1:], os.Stdout); err != nil {
		os.Exit(1)
	}
}
