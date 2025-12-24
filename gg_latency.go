package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	fullURL := "https://getgems.io/graphql/?operationName=nftSearch&variables=%7B%22query%22%3A%22%7B%5C%22%24and%5C%22%3A%5B%7B%5C%22collectionAddress%5C%22%3A%5C%22EQAOQdwdw8kGftJCSFgOErM1mBjYPe4DBPq8-AhF6vr9si5N%5C%22%7D%2C%7B%5C%22saleType%5C%22%3A%5C%22fix_price%5C%22%7D%5D%7D%22%2C%22attributes%22%3Anull%2C%22sort%22%3A%22%5B%7B%5C%22fixPrice%5C%22%3A%7B%5C%22order%5C%22%3A%5C%22asc%5C%22%7D%7D%2C%7B%5C%22index%5C%22%3A%7B%5C%22order%5C%22%3A%5C%22asc%5C%22%7D%7D%5D%22%2C%22count%22%3A10%7D&extensions=%7B%22clientLibrary%22%3A%7B%22name%22%3A%22%40apollo%2Fclient%22%2C%22version%22%3A%224.0.10%22%7D%2C%22persistedQuery%22%3A%7B%22version%22%3A1%2C%22sha256Hash%22%3A%22531556b37502a873b92ec74a0bc6cb411b186cd5413702def29064596b0b31f7%22%7D%7D"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var totalLatency int64
	var minLatency int64 = -1
	var maxLatency int64

	for i := 1; i <= 10; i++ {
		startTime := time.Now()

		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-gg-client", "v:1 l:ru s:mjk78bk4")

		resp, err := client.Do(req)
		latency := time.Since(startTime).Milliseconds()

		if err != nil {
		} else {
			statusCode := resp.StatusCode

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			totalLatency += latency

			if minLatency == -1 || latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}

			fmt.Printf("#%d %s Статус: %d | мс: %v\n", i, "", statusCode, latency)
			// fmt.Printf("Ответ:\n%s\n", string(body))
			// fmt.Println()
			_ = body
		}

		if i < 10 {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Printf("Средняя мс: %v\n", float64(totalLatency)/10.0)
}
