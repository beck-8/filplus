package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v2"
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

		// Proposal 定义 Proposal 结构体
		type Proposal struct {
			PieceCID             map[string]string `json:"PieceCID"`
			PieceSize            int64             `json:"PieceSize"`
			VerifiedDeal         bool              `json:"VerifiedDeal"`
			Client               string            `json:"Client"`
			Provider             string            `json:"Provider"`
			Label                string            `json:"Label"`
			StartEpoch           int64             `json:"StartEpoch"`
			EndEpoch             int64             `json:"EndEpoch"`
			StoragePricePerEpoch string            `json:"StoragePricePerEpoch"`
			ProviderCollateral   string            `json:"ProviderCollateral"`
			ClientCollateral     string            `json:"ClientCollateral"`
		}

		// State 定义 State 结构体
		type State struct {
			SectorNumber     int64 `json:"SectorNumber"`
			SectorStartEpoch int64 `json:"SectorStartEpoch"`
			LastUpdatedEpoch int64 `json:"LastUpdatedEpoch"`
			SlashEpoch       int64 `json:"SlashEpoch"`
		}

		// Entry 定义 result 中的键值对
		type Entry struct {
			Proposal Proposal `json:"Proposal"`
			State    State    `json:"State"`
		}

		// Response 定义整个 JSON 结构
		type Response struct {
			ID      int              `json:"id"`
			JSONRPC string           `json:"jsonrpc"`
			Result  map[string]Entry `json:"result"`
		}

		file, err := os.Open(ctx.String("file"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "打开文件失败: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		// 创建 json.Decoder
		decoder := json.NewDecoder(file)

		spDeal := map[string]map[string]int64{}
		fmt.Printf("%s ~ %s\n", ctx.String("start"), ctx.String("end"))

		w1 := tabwriter.NewWriter(os.Stdout, 18, 0, 4, ' ',
			0)
		fmt.Fprint(w1, "client\tsp\tdatacap(T)\n")
		w2 := tabwriter.NewWriter(os.Stdout, 18, 0, 4, ' ',
			0)
		fmt.Fprintf(w2, "ldn sum\t\tdatacap(T)\n")
		var totalDc int64

		// 读取 JSON 令牌
		for {
			// 读取下一个 JSON 令牌
			token, err := decoder.Token()
			if err == io.EOF {
				break // 文件结束
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "读取令牌失败: %v\n", err)
				os.Exit(1)
			}

			// 检查是否进入 result 对象
			if token == "result" {
				// 读取 result 对象的开始括号
				if _, err := decoder.Token(); err != nil {
					fmt.Fprintf(os.Stderr, "读取 result 开始括号失败: %v\n", err)
					os.Exit(1)
				}

				// 逐个处理 result 中的键值对
				for decoder.More() {
					// 读取键（例如 "100000000"）
					_, err := decoder.Token()
					if err != nil {
						fmt.Fprintf(os.Stderr, "读取键失败: %v\n", err)
						os.Exit(1)
					}

					// 解码对应的 Entry 对象
					var entry Entry
					if err := decoder.Decode(&entry); err != nil {
						fmt.Fprintf(os.Stderr, "解码 Entry 失败: %v\n", err)
						os.Exit(1)
					}

					provider := entry.Proposal.Provider
					client := entry.Proposal.Client
					pieceSize := entry.Proposal.PieceSize
					sectorStartEpoch := entry.Proposal.StartEpoch
					verified := entry.Proposal.VerifiedDeal

					sum := func() {
						if !pending {
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
					if !pending {
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
				}
			}
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
