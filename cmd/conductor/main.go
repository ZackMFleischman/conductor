package main

import (
	"context"
	"github.com/ZackMFleischman/conductor/internal/cli"
	"os"
)

func main() { os.Exit(cli.Run(context.Background(), cli.Env{}, os.Args[1:])) }
