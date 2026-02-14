package main

import (
	"context"
	"os"
)

func main() {
	ctx := context.Background()

	err := NewSession(os.Stdin, os.Stdout).Run(ctx)
	if err != nil {
		panic(err)
	}
}
