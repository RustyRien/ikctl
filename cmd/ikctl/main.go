// Command ikctl is the InfraKitchen command-line client.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rustyrien/ikctl/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := cli.Execute(ctx, cli.NewFactory(cli.SystemIO()), os.Args[1:])
	stop()
	os.Exit(code)
}
