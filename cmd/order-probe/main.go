package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// Ответ завёрнут в лишний "data": {"data":{"success":true,"data":{...}}}
type envelope struct {
	Data orderBookResp `json:"data"`
}

type orderBookResp struct {
	Success bool `json:"success"`
	Data    struct {
		AmtMaxBuyOrder  int64   `json:"amtMaxBuyOrder"`
		AmtMinSellOrder int64   `json:"amtMinSellOrder"`
		ECurrency       int     `json:"eCurrency"`
		CBuyOrders      int     `json:"cBuyOrders"`
		CSellOrders     int     `json:"cSellOrders"`
		Buy             []int64 `json:"rgCompactBuyOrders"`  // [цена, штук, цена, штук, ...] по убыванию
		Sell            []int64 `json:"rgCompactSellOrders"` // по возрастанию
	} `json:"data"`
}

func orderBookURL(hashName string) string {
	qp, _ := json.Marshal([]any{730, hashName})
	v := url.Values{
		"q":        {"Load"},
		"qp":       {string(qp)},
		"currency": {"1"}, // проверка: вдруг Steam вернёт доллары без авторизации
	}
	return "https://steamcommunity.com/market/orderbook?" + v.Encode()
}

func printLevels(title string, flat []int64, n int) {
	fmt.Println(title)
	if len(flat) < 2 {
		return
	}
	top := flat[0]
	var ahead int64
	for i := 0; i+1 < len(flat) && i/2 < n; i += 2 {
		price, qty := flat[i], flat[i+1]
		fmt.Printf("  %10.2f  x%-6d впереди %-7d %+6.1f%%\n",
			float64(price)/100, qty, ahead, (float64(price)/float64(top)-1)*100)
		ahead += qty
	}
}

func main() {
	name := "USP-S | Blueprint (Minimal Wear)"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest(http.MethodGet, orderBookURL(name), nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("status %d: %.200s", resp.StatusCode, body)
	}
	if err := os.WriteFile("orderbook.json", body, 0o644); err != nil {
		log.Fatal(err)
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		log.Fatalf("parse: %v", err)
	}
	ob := env.Data
	d := ob.Data

	fmt.Printf("success=%v currency=%d buy_orders=%d sell_orders=%d buy_levels=%d sell_levels=%d\n",
		ob.Success, d.ECurrency, d.CBuyOrders, d.CSellOrders, len(d.Buy)/2, len(d.Sell)/2)
	fmt.Printf("max buy %.2f | min sell %.2f | спред %.1f%%\n",
		float64(d.AmtMaxBuyOrder)/100, float64(d.AmtMinSellOrder)/100,
		(float64(d.AmtMinSellOrder)/float64(d.AmtMaxBuyOrder)-1)*100)
	printLevels("BUY (заявки):", d.Buy, 30)
	printLevels("SELL (лоты):", d.Sell, 5)
}
