package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/centrifugal/centrifuge-go"
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

	resp, err := http.Get("https://market.csgo.com/api/v2/get-ws-token?key=" + API_KEY)

	if err != nil {
	}
	defer resp.Body.Close()

	var tokenResp TokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		fmt.Println(err)
	}
	fmt.Println(tokenResp.Token)

	clientWs := centrifuge.NewJsonClient("wss://wsprice.csgo.com/connection/websocket", centrifuge.Config{Token: tokenResp.Token})
	defer clientWs.Close()

	// 3. Говорим, какой канал слушать, и что делать с каждым сообщением.
	sub, err := clientWs.NewSubscription("public:items:730:usd")
	if err != nil {
		log.Fatal(err)
	}
	sub.OnPublication(func(e centrifuge.PublicationEvent) {
		fmt.Println(string(e.Data)) // просто печатаем всё, что пришло
	})

	sub.Subscribe()

	// 4. Подключаемся.
	if err := clientWs.Connect(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("подключились, слушаем...")

	select {}
	// var wg sync.WaitGroup

	// wg.Add(2)
	// go func() {
	// 	defer wg.Done()
	// 	fmt.Println("Start test short prices")

	// 	for i := 0; i < 10000; i++ {
	// 		time.Sleep(5 * time.Second)

	// 		resp, err := http.Get("https://market.csgo.com/api/v2/prices/USD.json")
	// 		if err != nil {

	// 		}
	// 		defer resp.Body.Close()

	// 		var pricesShort PricesShort
	// 		if err := json.NewDecoder(resp.Body).Decode(&pricesShort); err != nil {

	// 		}
	// 		t := time.Unix(pricesShort.Time, 0)
	// 		fmt.Printf("SHORT - Unix - %d, MSK - %s\n", pricesShort.Time, t.Format(time.DateTime))

	// 	}
	// }()

	// go func() {
	// 	defer wg.Done()
	// 	fmt.Println("Start test long prices")

	// 	for i := 0; i < 10000; i++ {
	// 		time.Sleep(30 * time.Second)

	// 		resp, err := http.Get("https://market.csgo.com/api/full-export/USD.json")
	// 		if err != nil {

	// 		}
	// 		defer resp.Body.Close()

	// 		var pricesLong PricesLong
	// 		if err := json.NewDecoder(resp.Body).Decode(&pricesLong); err != nil {

	// 		}
	// 		t := time.Unix(pricesLong.Time, 0)
	// 		fmt.Printf("LONG - Unix - %d, MSK - %s\n", pricesLong.Time, t.Format(time.DateTime))
	// 	}
	// }()

	// wg.Wait()
}
