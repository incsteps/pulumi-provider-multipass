package main

import (
	"context"
	"fmt"
	"os"

	"github.com/incsteps/pulumi-provider-multipass/provider"
)

func main() {

	prov, err := provider.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}

	err = prov.Run(context.Background(), provider.Name, provider.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}
}
