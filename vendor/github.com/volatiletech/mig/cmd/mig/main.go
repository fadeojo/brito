package main

import (
	"fmt"
	"os"
)

const migVersion = "1.0.0"

func main() {
	// Too much happens between here and cobra's argument handling, for
	// something so simple. Just do it immediately.
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("mig v" + migVersion)
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
