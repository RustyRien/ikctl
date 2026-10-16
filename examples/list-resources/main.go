// Command list-resources shows how to use the InfraKitchen Go SDK directly.
//
// It reuses the credentials saved by 'ikctl login' (or IKCTL_SERVER/IKCTL_TOKEN):
//
//	go run ./examples/list-resources -name redis
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/config"
)

func main() {
	name := flag.String("name", "", "only resources whose name contains this string")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, *name)
	stop()
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, name string) error {
	client, err := sdk.NewFromConfig(config.Overrides{})
	if err != nil {
		return err
	}

	me, err := client.Auth.Me(ctx)
	if err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	fmt.Printf("Authenticated as %s on %s\n\n", me.Identifier, client.ServerURL())

	opts := sdk.ListOptions{Sort: &sdk.Sort{Field: "createdAt", Desc: true}, Limit: 50}
	if name != "" {
		opts.Filter = sdk.Like("name", name)
	}

	count := 0
	for res, err := range client.Resources.ListAll(ctx, opts) {
		if err != nil {
			return err
		}
		count++
		fmt.Printf("%-60s %-12s %s\n", res.String("name"), res.String("state"), res.String("status"))
	}
	fmt.Printf("\n%d resources\n", count)
	return nil
}
