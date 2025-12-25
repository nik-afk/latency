package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type FloorPriceItem struct {
	Name     string  `json:"name"`
	DefPrice float64 `json:"def_price"`
}

func main() {
	floorPrices := make(map[string]float64)
	floorData, err := os.ReadFile("floor_prices.json")
	if err == nil {
		var items []FloorPriceItem
		if err := json.Unmarshal(floorData, &items); err == nil {
			for _, item := range items {
				floorPrices[item.Name] = item.DefPrice
			}
			fmt.Printf("Загружено %d floor цен из floor_prices.json\n", len(floorPrices))
		}
	}

	fullURL := "https://getgems.io/graphql/?operationName=nftSearch&variables=%7B%22query%22%3A%22%7B%5C%22%24and%5C%22%3A%5B%7B%5C%22collectionAddressList%5C%22%3A%5B%5C%22EQAbfjxb1uxz66R_c6sjdYysf7kuERaRAvcDXIYfSHWTRwuz%5C%22%5D%7D%2C%7B%5C%22saleType%5C%22%3A%5C%22fix_price%5C%22%7D%5D%7D%22%2C%22attributes%22%3Anull%2C%22sort%22%3A%22%5B%7B%5C%22createdAt%5C%22%3A%7B%5C%22order%5C%22%3A%5C%22desc%5C%22%7D%7D%5D%22%2C%22count%22%3A5%7D&extensions=%7B%22clientLibrary%22%3A%7B%22name%22%3A%22%40apollo%2Fclient%22%2C%22version%22%3A%224.0.10%22%7D%2C%22persistedQuery%22%3A%7B%22version%22%3A1%2C%22sha256Hash%22%3A%22531556b37502a873b92ec74a0bc6cb411b186cd5413702def29064596b0b31f7%22%7D%7D"

	ips := []string{
		"146.19.119.141",
		"146.19.119.220",
		"146.19.119.62",
		"146.19.119.63",
		"146.19.119.64",
		"146.19.119.65",
		"185.35.137.17",
		"185.35.137.18",
		"185.35.137.19",
	}

	newHTTPClientWithSourceIP := func(sourceIP string) *http.Client {
		dialer := &net.Dialer{
			LocalAddr: &net.TCPAddr{IP: net.ParseIP(sourceIP), Port: 0},
		}
		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := dialer.DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}
				return conn, nil
			},
		}
		return &http.Client{
			Transport: transport,
			Timeout:   80 * time.Millisecond,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		}
	}

	fmt.Printf("🌐 Доступно %d IP адресов для парсинга\n", len(ips))

	file, err := os.OpenFile("profitable_items.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	var (
		totalRequests      int64
		responseTimes      []int64
		responseStatuses   []int
		lastErrorStatus    int
		mu                 sync.Mutex
		fileMu             sync.Mutex
		recentItems              = make(map[string]time.Time)
		currentIPIndex     int64 = 0
		sharedRequestCount int64 = 0
	)

	makeRequestWithClient := func(floorPricesMap map[string]float64, client *http.Client, currentIP string, workerID int) {
		startTime := time.Now()

		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-gg-client", "v:1 l:ru s:mjbvqc5i")

		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		statusCode := resp.StatusCode

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		elapsed := time.Since(startTime).Milliseconds()

		mu.Lock()
		totalRequests++
		responseTimes = append(responseTimes, elapsed)
		responseStatuses = append(responseStatuses, statusCode)
		if statusCode != 200 {
			lastErrorStatus = statusCode
		}
		mu.Unlock()

		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			if data, ok := jsonData.(map[string]interface{}); ok {
				if alphaNft, ok := data["data"].(map[string]interface{}); ok {
					if search, ok := alphaNft["alphaNftItemSearch"].(map[string]interface{}); ok {
						if edges, ok := search["edges"].([]interface{}); ok {
							for _, edge := range edges {
								if edgeMap, ok := edge.(map[string]interface{}); ok {
									if node, ok := edgeMap["node"].(map[string]interface{}); ok {
										if sale, ok := node["sale"].(map[string]interface{}); ok {
											if typename, ok := sale["__typename"].(string); ok && typename == "NftSaleFixPrice" {
												name := ""
												if collection, ok := node["collection"].(map[string]interface{}); ok {
													if nameVal, ok := collection["name"].(string); ok {
														name = nameVal
													}
												}

												if fullPriceVal, ok := sale["fullPrice"].(string); ok {
													if priceNano, err := strconv.ParseInt(fullPriceVal, 10, 64); err == nil {
														priceTON := float64(priceNano) / 1_000_000_000.0

														if floorPrice, exists := floorPricesMap[name]; exists && floorPrice > 0 {
															if priceTON < floorPrice {
																itemKey := fmt.Sprintf("%s_%.2f", name, priceTON)

																fileMu.Lock()
																now := time.Now()
																if lastTime, exists := recentItems[itemKey]; !exists || now.Sub(lastTime) > 30*time.Second {
																	timestamp := now.Format("2006-01-02 15:04:05")
																	line := fmt.Sprintf("[%s] 🎯 ВЫГОДНЫЙ ПРЕДМЕТ: %s | Цена: %.2f TON | Floor: %.2f TON | Экономия: %.2f TON\n",
																		timestamp, name, priceTON, floorPrice, floorPrice-priceTON)
																	file.WriteString(line)
																	file.Sync()
																	recentItems[itemKey] = now

																	for key, t := range recentItems {
																		if now.Sub(t) > 5*time.Minute {
																			delete(recentItems, key)
																		}
																	}
																}
																fileMu.Unlock()
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	var (
		sharedIPMutex sync.RWMutex
		sharedIP      = ips[0]
		sharedClient  = newHTTPClientWithSourceIP(sharedIP)
	)

	fmt.Printf("Запуск 2 потоков на IP: %s\n", sharedIP)

	for i := 0; i < 2; i++ {
		go func(workerID int) {
			for {
				sharedIPMutex.RLock()
				client := sharedClient
				ip := sharedIP
				sharedIPMutex.RUnlock()

				makeRequestWithClient(floorPrices, client, ip, workerID)

				requestCount := atomic.AddInt64(&sharedRequestCount, 1)

				if requestCount%200 == 0 {
					sharedIPMutex.Lock()
					oldIP := sharedIP
					currentIPIdx := atomic.AddInt64(&currentIPIndex, 1) % int64(len(ips))
					sharedIP = ips[currentIPIdx]
					sharedClient = newHTTPClientWithSourceIP(sharedIP)
					fmt.Printf("[W %d] 🔄 Ротация IP каждые 200 запросов: %s → %s (всего запросов: %d)\n", workerID, oldIP, sharedIP, requestCount)
					sharedIPMutex.Unlock()
				}

				time.Sleep(1 * time.Second)
			}
		}(i)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fmt.Println("Запуск парсера...\n")

	for range ticker.C {
		mu.Lock()
		total := totalRequests
		times := make([]int64, len(responseTimes))
		copy(times, responseTimes)
		statuses := make([]int, len(responseStatuses))
		copy(statuses, responseStatuses)
		errorStatus := lastErrorStatus
		responseTimes = []int64{}
		responseStatuses = []int{}
		if errorStatus != 0 {
			lastErrorStatus = 0
		}
		mu.Unlock()

		var avgTime float64
		if len(times) > 0 {
			var sum int64
			for _, t := range times {
				sum += t
			}
			avgTime = float64(sum) / float64(len(times))
		}

		lastStatus := 0
		if len(statuses) > 0 {
			lastStatus = statuses[len(statuses)-1]
		}

		fmt.Printf("Запрос: %d | Статус: %d | ср мс: %.2f\n", total, lastStatus, avgTime)
	}
}
