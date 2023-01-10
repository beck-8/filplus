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
	Name:  "query",
	Usage: "query datacap",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "sp",
			Value:    "",
			Usage:    "specify sp list",
			Required: false,
			Aliases:  []string{"s"},
		},
		&cli.StringFlag{
			Name:     "client",
			Value:    "",
			Usage:    "specify client id list",
			Required: true,
			Aliases:  []string{"c"},
		},
		&cli.BoolFlag{
			Name:    "lookup",
			Value:   true,
			Usage:   "show address id",
			Aliases: []string{"l"},
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
		w := tabwriter.NewWriter(os.Stdout, 18, 0, 4, ' ',
			0)
		fmt.Fprint(w, "client\tsp\tdatacap(T)\n")

		var totalDc float64
		for _, client := range clients {
			id, err := StateLookupID(client)
			if err != nil {
				return err
			}

			if body, err := getDc(id); err != nil {
				return err
			} else {
				for _, stat := range body.Stats {
					for _, sp := range sps {
						if stat.Provider == sp {

							dc, err := strconv.ParseFloat(stat.TotalDealSize, 64)
							if err != nil {
								return err
							}
							if ctx.Bool("lookup") {
								fmt.Fprintf(w, "%s\t%v\t%v\n", id, sp, dc/(1<<40))
							} else {
								fmt.Fprintf(w, "%s\t%v\t%v\n", client, sp, dc/(1<<40))
							}

							totalDc += dc
						}
					}
				}
			}

		}
		fmt.Fprintf(w, "Total Datacap:\t\t%v\n", totalDc/(1<<40))
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
var glifUrl = "https://api.node.glif.io/rpc/v0"

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

func StateLookupID(addr string) (string, error) {
	payload := strings.NewReader(fmt.Sprintf("{\n  \"jsonrpc\": \"2.0\",\n  \"method\": \"Filecoin.StateLookupID\",\n  \"params\": [\n  \"%s\",\n  [\n  ]\n],\n  \"id\": 1\n}", addr))

	req, err := http.NewRequest("POST", glifUrl, payload)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	} else if response.StatusCode != 200 {
		return "", fmt.Errorf("client %s LookupID return code is %v", addr, response.StatusCode)
	}
	defer response.Body.Close()

	var bd map[string]interface{}
	if body, err := io.ReadAll(response.Body); err != nil {
		return "", err
	} else {
		err := json.Unmarshal(body, &bd)
		if err != nil {
			return "", err
		}
	}
	return bd["result"].(string), nil
}
