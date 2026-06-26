package main

import (
	"os"

	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/app"
)

func main() {
	os.Exit(app.New().Run(os.Args[1:]))
}
