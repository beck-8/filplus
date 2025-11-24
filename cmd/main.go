package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/klauspost/compress/zstd"
)

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

func main() {
	// 打开 zstd 压缩文件
	file, err := os.Open("D:/tmp/StateMarketDeals.json.zst")
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开文件失败: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 创建 zstd 解压器
	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建zstd解压器失败: %v\n", err)
		os.Exit(1)
	}
	defer zstdReader.Close()

	// 创建 json.Decoder（从zstd解压流读取）
	decoder := json.NewDecoder(zstdReader)

	// zstd文件结构直接是 map[string]Entry
	// 读取开始的 '{'
	if _, err := decoder.Token(); err != nil {
		fmt.Fprintf(os.Stderr, "读取开始括号失败: %v\n", err)
		os.Exit(1)
	}

	// 逐个处理键值对
	for decoder.More() {
		// 读取键（例如 "100000000"）
		key, err := decoder.Token()
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

		// 筛选后续逻辑
		// 筛选 EndEpoch < 5098759
		if entry.Proposal.Provider == "f03091738" || entry.Proposal.Provider == "f0xx" {
			d, _ := json.Marshal(entry)
			fmt.Println(key, string(d))
			// 可选：将结果写入文件
			// writeToFile(key, entry)
		}
	}

	// 以下是旧的处理 Response 结构（包含 id, jsonrpc, result）的逻辑，已不再需要
	// // 读取 JSON 令牌
	// for {
	// 	// 读取下一个 JSON 令牌
	// 	token, err := decoder.Token()
	// 	if err == io.EOF {
	// 		break // 文件结束
	// 	}
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "读取令牌失败: %v\n", err)
	// 		os.Exit(1)
	// 	}
	//
	// 	// 检查是否进入 result 对象
	// 	if token == "result" {
	// 		// 读取 result 对象的开始括号
	// 		if _, err := decoder.Token(); err != nil {
	// 			fmt.Fprintf(os.Stderr, "读取 result 开始括号失败: %v\n", err)
	// 			os.Exit(1)
	// 		}
	//
	// 		// 逐个处理 result 中的键值对
	// 		for decoder.More() {
	// 			// 读取键（例如 "100000000"）
	// 			_, err := decoder.Token()
	// 			if err != nil {
	// 				fmt.Fprintf(os.Stderr, "读取键失败: %v\n", err)
	// 				os.Exit(1)
	// 			}
	//
	// 			// 解码对应的 Entry 对象
	// 			var entry Entry
	// 			if err := decoder.Decode(&entry); err != nil {
	// 				fmt.Fprintf(os.Stderr, "解码 Entry 失败: %v\n", err)
	// 				os.Exit(1)
	// 			}
	//
	// 			// 筛选后续逻辑
	// 			if entry.Proposal.Provider == "f03091738" {
	// 				d, _ := json.Marshal(entry)
	// 				fmt.Println(string(d))
	// 			}
	// 		}
	// 	}
	// }
}

// 可选：将结果写入文件的函数
func writeToFile(key interface{}, entry Entry) {
	outputFile, err := os.OpenFile("output.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开输出文件失败: %v\n", err)
		return
	}
	defer outputFile.Close()

	// 格式化输出
	// data := map[string]interface{}{"key": key, "entry": entry}
	jsonData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化输出失败: %v\n", err)
		return
	}

	if _, err := outputFile.WriteString(string(jsonData) + "\n"); err != nil {
		fmt.Fprintf(os.Stderr, "写入文件失败: %v\n", err)
	}
}
