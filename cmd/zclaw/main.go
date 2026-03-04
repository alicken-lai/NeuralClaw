package main

import (
	"log"

	"neuralclaw/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
