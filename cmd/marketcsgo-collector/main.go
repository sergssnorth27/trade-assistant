package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/names"
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/snapshot"
)

const API_KEY = "003jE2R73Kra436S115LPgC25X2E1yC"

type PricesShort struct {
	Success  bool        `json:"success"`
	Time     int64       `json:"time"`
	Currency string      `json:"currency"`
	Items    []PriceItem `json:"items"`
}

type PriceItem struct {
	MarketHashName string `json:"market_hash_name"`
	Volume         int    `json:"volume,string"` // ",string" сам распарсит "31" → 31
	Price          Price  `json:"price"`         // свой тип, см. ниже
}

type PricesLong struct {
	Success  bool   `json:"success"`
	Time     int64  `json:"time"`
	Currency string `json:"currency"`
}

type TokenResp struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
}

type Price int64

func main() {
	ctx := context.Background()
	//wsEventChan := make(chan ws.Event, 1000)
	// infoChan := make(chan snapshot.Info)
	infoChan := make(chan snapshot.Info)

	// go func() {
	// 	ws.Run(ctx, API_KEY, wsEventChan)
	// }()

	// for event := range wsEventChan {
	// 	fmt.Println(event)
	// }
	dict := names.Dict{}
	go func() {
		dict.Load(ctx)
	}()

	go func() {
		snapshot.Run(ctx, API_KEY, infoChan)
	}()

	for info := range infoChan {
		nowTime := time.Now().Local().Format("3:04:05")
		check := time.Unix(info.Time, 0).Local().Format("3:04:05")
		fmt.Printf("now - %s / check - %s, %d\n", nowTime, check, info.CountItem)
	}
}
