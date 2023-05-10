package main

import (
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

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
		&cli.BoolFlag{
			Name:  "sum",
			Value: false,
			Usage: "summarize the LDN quota",
		},
		&cli.BoolFlag{
			Name:  "pending",
			Value: false,
			Usage: "include pending：after publish",
		},
	},
	Action: func(ctx *cli.Context) error {
		var startEpoch, endEpoch int64
		var err error

		pending := ctx.Bool("pending")
		if pending {
			if ctx.IsSet("start") || ctx.IsSet("end") {
				return fmt.Errorf("the pending parameter is not allowed to be used with start or end")
			}
		}

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
		clientsLen := len(clients)
		spsLen := len(sps)

		file := ctx.String("file")
		f, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		spDeal := map[string]map[string]int64{}
		fmt.Printf("%s ~ %s\n", ctx.String("start"), ctx.String("end"))

		w1 := tabwriter.NewWriter(os.Stdout, 18, 0, 4, ' ',
			0)
		fmt.Fprint(w1, "client\tsp\tdatacap(T)\n")
		w2 := tabwriter.NewWriter(os.Stdout, 18, 0, 4, ' ',
			0)
		fmt.Fprintf(w2, "ldn sum\t\tdatacap(T)\n")
		var totalDc int64

		//其他json解析方式性能低下，使用jsonparser库8.3G文件花费49s，原生花费4m；python3 原生 3m+,orjson 2m48s。
		err = jsonparser.ObjectEach(f, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			provider, err := jsonparser.GetString(value, "Proposal", "Provider")
			if err != nil {
				return err
			}
			client, err := jsonparser.GetString(value, "Proposal", "Client")
			if err != nil {
				return err
			}
			pieceSize, err := jsonparser.GetInt(value, "Proposal", "PieceSize")
			if err != nil {
				return err
			}
			sectorStartEpoch, err := jsonparser.GetInt(value, "State", "SectorStartEpoch")
			if err != nil {
				return err
			}
			verified, err := jsonparser.GetBoolean(value, "Proposal", "VerifiedDeal")
			if err != nil {
				return err
			}

			sum := func() {
				if pending == false {
					if sectorStartEpoch >= startEpoch && sectorStartEpoch <= endEpoch {
						if _, ok := spDeal[client]; ok {
							spDeal[client][provider] += pieceSize
						} else {
							spDeal[client] = map[string]int64{}
							spDeal[client][provider] += pieceSize
						}
						totalDc += pieceSize
					}
				} else {
					if _, ok := spDeal[client]; ok {
						spDeal[client][provider] += pieceSize
					} else {
						spDeal[client] = map[string]int64{}
						spDeal[client][provider] += pieceSize
					}
					totalDc += pieceSize
				}

			}

			var judgment bool
			if pending == false {
				judgment = verified && sectorStartEpoch != -1
			} else {
				judgment = verified
			}
			if judgment {
				if spsLen != 0 && clientsLen != 0 {
					if ContainsInMap(sps, provider) && ContainsInMap(clients, client) {
						sum()
					}
				} else {
					if ContainsInMap(sps, provider) || ContainsInMap(clients, client) {
						sum()
					}
				}
			}

			return nil
		}, "result")
		if err != nil {
			return err
		}

		if clientsLen != 0 && spsLen != 0 {
			for _, client := range strings.Split(ctx.String("client"), ",") {
				if _, ok := spDeal[client]; ok {
					var sumPiecesize int64 = 0
					for _, sp := range strings.Split(ctx.String("sp"), ",") {
						if piecesize, ok := spDeal[client][sp]; ok {
							sumPiecesize += piecesize
							fmt.Fprintf(w1, "%s\t%s\t%v\n", client, sp, float64(piecesize)/(1<<40))
						}
					}
					fmt.Fprintf(w2, "%s\t\t%v\n", client, float64(sumPiecesize)/(1<<40))
				}
			}
		} else {
			for client, v := range spDeal {
				var sumPiecesize int64 = 0
				for sp, piecesize := range v {
					sumPiecesize += piecesize
					fmt.Fprintf(w1, "%s\t%s\t%v\n", client, sp, float64(piecesize)/(1<<40))
				}
				fmt.Fprintf(w2, "%s\t\t%v\n", client, float64(sumPiecesize)/(1<<40))
			}
		}

		fmt.Fprintf(w1, "Total Datacap\t\t%v\n\n", float64(totalDc)/(1<<40))
		w1.Flush()
		if ctx.Bool("sum") {
			w2.Flush()
		}
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
		if v == "" {
			continue
		}
		set[v] = struct{}{}
	}
	return set
}

// ContainsInMap 判断字符串是否在 map 中
func ContainsInMap(m map[string]struct{}, s string) bool {
	_, ok := m[s]
	return ok
}
