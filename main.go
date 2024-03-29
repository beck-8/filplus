package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {

	app := &cli.App{
		Name:    "filplus",
		Usage:   "A tool to query datacap",
		Version: UserVersion(),
		Commands: []*cli.Command{
			query,
			sum,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("ERROR: %+v\n", err)
		os.Exit(1)
		return
	}
}
