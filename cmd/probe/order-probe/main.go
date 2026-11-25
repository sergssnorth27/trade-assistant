// Проба стакана Steam через авторизованный клиент (должна прийти валюта USD).
//
//	go run ./cmd/order-probe
//	go run ./cmd/order-probe "Gamma Case"
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/steam"
	"github.com/sergssnorth27/trade-assistant/internal/steam/auth"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file: %v", err)
	}

	name := "AK-47 | Slate (Field-Tested)"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// ← как в steam-history: логин и авторизованный клиент.
	// Токены, пароль и Guard-код НЕ печатаем.
	session, err := auth.Login(ctx, &http.Client{Timeout: 30 * time.Second},
		os.Getenv("STEAM_LOGIN"), os.Getenv("STEAM_PASSWORD"), os.Getenv("STEAM_SHARED_SECRET"))
	if err != nil {
		log.Fatalf("steam login: %v", err)
	}
	client, err := steam.NewAuthorizedClient(session)
	if err != nil {
		log.Fatalf("steam client: %v", err)
	}
	// конец блока авторизации

	start := time.Now()
	book, err := client.OrderBook(ctx, name)
	if err != nil {
		log.Fatalf("order book: %v", err)
	}
	fmt.Printf("%s (%s)\n", name, time.Since(start).Round(time.Millisecond))
	fmt.Printf("max buy %.3f | min sell %.3f | спред %.1f%%\n",
		dollars(book.MaxBuyOrder), dollars(book.MinSellOrder),
		(float64(book.MinSellOrder)/float64(book.MaxBuyOrder)-1)*100)
	fmt.Printf("заявок: %d штук на %d уровнях | лотов: %d штук на %d уровнях\n",
		book.QtyBuyOrders, len(book.BuyOrders), book.QtySellOrders, len(book.SellOrders))

	fmt.Println("BUY (по убыванию):")
	printLevels(book.BuyOrders, 10, book.MaxBuyOrder)
	fmt.Println("SELL (по возрастанию):")
	printLevels(book.SellOrders, 5, book.MaxBuyOrder)

	// Самопроверка разбора.
	if len(book.BuyOrders) > 0 && book.BuyOrders[0].Price != book.MaxBuyOrder {
		fmt.Println("⚠ первый уровень заявок не совпадает с MaxBuyOrder")
	}
	if len(book.SellOrders) > 0 && book.SellOrders[0].Price != book.MinSellOrder {
		fmt.Println("⚠ первый уровень лотов не совпадает с MinSellOrder")
	}
	var sum int64
	for _, l := range book.BuyOrders {
		sum += l.Count
	}
	if sum != book.QtyBuyOrders {
		fmt.Printf("⚠ сумма штук по уровням %d ≠ QtyBuyOrders %d\n", sum, book.QtyBuyOrders)
	}
}

func dollars(p domain.Price) float64 { return float64(p) / 1000 }

func printLevels(levels []steam.OrderLevel, n int, top domain.Price) {
	var ahead int64
	for i, l := range levels {
		if i >= n {
			break
		}
		fmt.Printf("  %8.3f  x%-6d впереди %-6d %+6.1f%%\n",
			dollars(l.Price), l.Count, ahead, (float64(l.Price)/float64(top)-1)*100)
		ahead += l.Count
	}
}
