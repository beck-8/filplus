package main

import (
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-state-types/builtin/v9/market"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

type MarketDeal struct {
	Proposal market.DealProposal
	State    market.DealState
}

type Deal struct {
	JsonRpc string                 `json:"jsonrpc"`
	Result  map[string]*MarketDeal `json:"result"`
	Id      int                    `json:"id"`
}

var sum = &cli.Command{
	Name:  "calculate",
	Usage: "from file calculate datacap",
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
			Required: false,
			Aliases:  []string{"c"},
		},
		&cli.StringFlag{
			Name:     "file",
			Value:    "",
			Usage:    "specify deal file",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "start",
			Value: "2020-08-25 06:00:00",
			Usage: "specify start time",
		},
		&cli.StringFlag{
			Name:  "end",
			Value: "2060-08-25 06:00:00",
			Usage: "specify end time",
		},
	},
	Action: func(ctx *cli.Context) error {
		var startEpoch, endEpoch int64
		var err error

		if start := ctx.String("start"); start != "" {
			startEpoch, err = timeToHeight(start)
			if err != nil {
				return err
			}
		}
		if end := ctx.String("end"); end != "" {
			endEpoch, err = timeToHeight(end)
			if err != nil {
				return err
			}
		}

		clients := ConvertStrSlice2Map(strings.Split(ctx.String("client"), ","))
		sps := ConvertStrSlice2Map(strings.Split(ctx.String("sp"), ","))

		file := ctx.String("file")
		deal := Deal{}
		f, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		err = json.Unmarshal(f, &deal)
		if err != nil {
			return err
		}

		sp_deal := map[string]map[string]int64{}
		fmt.Printf("%s ~ %s\n", ctx.String("start"), ctx.String("end"))

		w := tabwriter.NewWriter(os.Stdout, 18, 0, 4, ' ',
			0)
		fmt.Fprint(w, "client\tsp\tdatacap(T)\n")
		var totalDc int64

		for _, v := range deal.Result {

			provider := v.Proposal.Provider.String()
			client := v.Proposal.Client.String()
			pieceSize := v.Proposal.PieceSize
			sectorStartEpoch := int64(v.State.SectorStartEpoch)

			if v.Proposal.VerifiedDeal && sectorStartEpoch != -1 {
				if !ContainsInMap(sps, provider) && !ContainsInMap(clients, client) {
					continue
				}
				if sectorStartEpoch >= startEpoch && sectorStartEpoch <= endEpoch {
					if _, ok := sp_deal[provider]; ok {
						sp_deal[provider][client] += int64(pieceSize)
					} else {
						sp_deal[provider] = map[string]int64{}
						sp_deal[provider][client] += int64(pieceSize)
					}
					totalDc += int64(pieceSize)
				}

			}

		}

		for sp, v := range sp_deal {
			for client, piecesize := range v {
				fmt.Fprintf(w, "%s\t%s\t%v\n", client, sp, piecesize/(1<<40))
			}
		}

		fmt.Fprintf(w, "Total Datacap\t\t%v\n", totalDc/(1<<40))
		w.Flush()
		return nil

	},
}

func timeToHeight(text string) (int64, error) {
	// 主网启动时间
	bootstrapTime := int64(1598306400)
	// 中国时区
	loc, _ := time.LoadLocation("PRC")
	stamp, err := time.ParseInLocation("2006-1-2 15:04:05", text, loc)
	if err != nil {
		return 0, err
	}
	return (stamp.Unix() - bootstrapTime) / 30, nil
}

// ConvertStrSlice2Map 将字符串 slice 转为 map[string]struct{}
func ConvertStrSlice2Map(sl []string) map[string]struct{} {
	set := make(map[string]struct{}, len(sl))
	for _, v := range sl {
		set[v] = struct{}{}
	}
	return set
}

// ContainsInMap 判断字符串是否在 map 中
func ContainsInMap(m map[string]struct{}, s string) bool {
	_, ok := m[s]
	return ok
}
