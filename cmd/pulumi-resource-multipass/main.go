package main

import (
	"log"

	"github.com/incsteps/pulumi-provider-multipass/provider"
)

func main() {
	if err := provider.Build().BuildAndRun(); err != nil {
		log.Fatal(err)
	}
}
