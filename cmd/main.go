package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	json "github.com/goccy/go-json"
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

// NDJSONEntry NDJSON 格式的条目
type NDJSONEntry struct {
	DealID   int64    `json:"DealID"`
	Proposal Proposal `json:"Proposal"`
	State    State    `json:"State"`
}

// 筛选条件函数类型
type FilterFunc func(entry *NDJSONEntry) bool

func main() {
	// 命令行参数
	// https://marketdeals.s3.ap-northeast-1.amazonaws.com/StateMarketDeals.ndjson.zst
	inputFile := flag.String("input", "D:/tmp/StateMarketDeals.ndjson.zst", "输入文件路径 (zstd 压缩的 NDJSON)")
	workers := flag.Int("workers", runtime.NumCPU(), "并发 worker 数量")
	provider := flag.String("provider", "", "筛选指定 Provider (可选)")
	flag.Parse()

	// 打开 zstd 压缩文件
	file, err := os.Open(*inputFile)
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

	// 构建筛选函数
	filter := buildFilter(*provider)

	// 使用并发处理
	processWithWorkers(zstdReader, *workers, filter)
}

// buildFilter 构建筛选函数
func buildFilter(provider string) FilterFunc {
	return func(entry *NDJSONEntry) bool {
		// 如果指定了 provider，只匹配该 provider
		if provider != "" {
			return entry.Proposal.Provider == provider
		}
		// 默认筛选条件
		return entry.Proposal.Provider == "f03091738" || entry.Proposal.Provider == "f0xx"
	}
}

// processWithWorkers 使用 worker pool 并发处理
func processWithWorkers(reader *zstd.Decoder, numWorkers int, filter FilterFunc) {
	// 行读取 channel
	lines := make(chan []byte, numWorkers*100)
	// 结果 channel
	results := make(chan string, numWorkers*100)

	var wg sync.WaitGroup
	var processedCount atomic.Int64
	var matchedCount atomic.Int64

	// 启动 worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lines {
				processedCount.Add(1)

				var entry NDJSONEntry
				if err := json.Unmarshal(line, &entry); err != nil {
					// 跳过解析失败的行
					continue
				}

				// 应用筛选条件
				if filter(&entry) {
					matchedCount.Add(1)
					// 重新序列化输出
					output, _ := json.Marshal(entry)
					results <- string(output)
				}
			}
		}()
	}

	// 启动结果输出 goroutine
	var outputWg sync.WaitGroup
	outputWg.Add(1)
	go func() {
		defer outputWg.Done()
		for result := range results {
			fmt.Println(result)
		}
	}()

	// 读取文件并分发给 workers
	scanner := bufio.NewScanner(reader)
	// 增大缓冲区以处理长行
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		// 复制行数据，因为 scanner.Bytes() 返回的切片会被复用
		line := make([]byte, len(scanner.Bytes()))
		copy(line, scanner.Bytes())
		lines <- line
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "读取文件错误: %v\n", err)
	}

	// 关闭 lines channel，等待 workers 完成
	close(lines)
	wg.Wait()

	// 关闭 results channel，等待输出完成
	close(results)
	outputWg.Wait()

	// 输出统计信息
	fmt.Fprintf(os.Stderr, "\n处理完成: 共处理 %d 条, 匹配 %d 条\n",
		processedCount.Load(), matchedCount.Load())
}
