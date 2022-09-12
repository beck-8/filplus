package main

import (
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
)

var query = &cli.Command{
	Name:        "query",
	Description: "query datacap",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "sp",
			Value:    "f01877571,f01880047,f01882184,f01878005,f01882177",
			Usage:    "Specify SP List",
			Required: false,
			Aliases:  []string{"s"},
		},
		&cli.StringFlag{
			Name:     "client",
			Value:    "",
			Usage:    "Specify Client ID List",
			Required: true,
			Aliases:  []string{"c"},
		},
	},
	Action: func(ctx *cli.Context) error {
		clients := strings.Split(ctx.String("client"), ",")
		if clients[0] == "" {
			return fmt.Errorf("please specify correct client address")
		}

		sps := strings.Split(ctx.String("sp"), ",")
		if sps[0] == "" {
			return fmt.Errorf("please specify correct sp address")
		}
		w := tabwriter.NewWriter(os.Stdout, 15, 4, 1, ' ',
			0)
		fmt.Fprint(w, "client\tsp\tdatacap(T)\n")

		var totalDc float64
		for _, client := range clients {

			if body, err := getDc(client); err != nil {
				return err
			} else {
				for _, stat := range body.Stats {
					for _, sp := range sps {
						if stat.Provider == sp {

							dc, err := strconv.ParseFloat(stat.TotalDealSize, 64)
							if err != nil {
								return err
							}
							fmt.Fprintf(w, "%s\t%v\t%v\n", client, sp, dc/(1<<40))

							totalDc += dc
						}
					}
				}
			}

		}
		fmt.Fprintf(w, "Total Datacap:\t%v\n", totalDc/(1<<40))
		w.Flush()
		return nil

	},
}

type Body struct {
	Stats     []Stats `json:"stats"`
	Name      string  `json:"name"`
	DealCount string  `json:"dealCount"`
}
type Stats struct {
	Provider      string `json:"provider"`
	TotalDealSize string `json:"total_deal_size"`
	Percent       string `json:"percent"`
}

var url = "https://api.filplus.d.interplanetary.one/api/getDealAllocationStats/"

func getDc(client string) (Body, error) {

	bd := Body{}

	response, err := http.Get(url + client)
	if err != nil {
		return bd, err
	} else if response.StatusCode != 200 {
		return bd, fmt.Errorf("client %s query return code is %v", client, response.StatusCode)
	}
	defer response.Body.Close()

	if body, err := io.ReadAll(response.Body); err != nil {
		return bd, err
	} else {
		err := json.Unmarshal(body, &bd)
		if err != nil {
			return bd, err
		}
	}

	return bd, nil
}
