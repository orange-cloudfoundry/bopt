package main

import (
	"github.com/orange-cloudfoundry/bopt/cmd"
	"os"
	"fmt"
)

func main() {
	err := cmd.Parse(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bopt: %s\n", err.Error())
		os.Exit(1)
	}
}
