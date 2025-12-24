package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func main() {
	apiURL := "https://fragment.com/api?hash=9e90a91b0a1e35b5b0"

	formData := url.Values{}
	formData.Set("hash", "9e90a91b0a1e35b5b0")
	formData.Set("type", "numbers")
	formData.Set("sort", "price_asc")
	formData.Set("filter", "sale")
	formData.Set("query", "")
	formData.Set("method", "searchAuctions")

	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	var totalLatency time.Duration
	var successCount int

	for i := 1; i <= 10; i++ {
		requestTime := time.Now()
		req, err := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		req.Header.Set("Cookie", "stel_ssid=38a07a823fcd10486b_15405110538129332560; stel_token=6de8178117fb3ea00514fbe0c1ae39d16de8179a6de815f80e9d60d53a700b6b5ecce; stel_dt=-60")

		startTime := time.Now()
		resp, err := client.Do(req)
		responseTime := time.Since(startTime)

		if err != nil {
			continue
		}

		_, err = io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			continue
		}

		successCount++
		totalLatency += responseTime
		fmt.Printf("#%d %s Статус: %d | мс: %v\n",
			i, requestTime.Format("15:04:05.000"), resp.StatusCode, responseTime)
		// fmt.Printf("Ответ: %s\n", string(body))

		if i < 10 {
			time.Sleep(1 * time.Second)
		}
	}

	if successCount > 0 {
		avgLatency := totalLatency / time.Duration(successCount)
		fmt.Printf("Средняя мс: %v\n", avgLatency)
	}
}
