package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/joho/godotenv"
	"github.com/sergssnorth27/trade-assistant/internal/steam/auth"
)

var items = []string{
	"M4A4 | Temukau (Field-Tested)",
	"AK-47 | Nightwish (Field-Tested)",
	"AK-47 | Redline (Field-Tested)",
	"AWP | Asiimov (Field-Tested)",
	"Desert Eagle | Printstream (Field-Tested)",
	"USP-S | Kill Confirmed (Field-Tested)",
	"M4A1-S | Printstream (Field-Tested)",
	"Glock-18 | Water Elemental (Field-Tested)",
}

var itemsID = []string{
	"G181020CC093004",
	"G181020CB073004",
	"G18102080043004",
	"G181020CC043004",
	"G181020C9083004",
	"G1810208D093004",
	"G181020B7053004",
	"G181020C90A3004",
	"G181020E0033004",
}

func buildURL(endpoint, name string) (string, error) {
	switch endpoint {
	case "page":
		return "https://steamcommunity.com/market/listings/730/" + url.PathEscape(name) + "?l=english", nil
	case "orderbook":
		qp, _ := json.Marshal([]any{730, name})
		return "https://steamcommunity.com/market/orderbook?q=Load&qp=" + url.QueryEscape(string(qp)), nil
	case "overview":
		return "https://steamcommunity.com/market/priceoverview/?appid=730&currency=1&market_hash_name=" + url.QueryEscape(name), nil
	}
	return "", fmt.Errorf("unknown endpoint %q (page, orderbook, overview)", endpoint)
}

// checkAuth проверяет, что Steam видит нас вошедшими.
func checkAuth(client *http.Client, cookie string) {
	get := func(u string) []byte {
		req, _ := http.NewRequest(http.MethodGet, u, nil)
		req.Header.Set("Cookie", "steamLoginSecure="+cookie)
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("check auth: %v", err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return b
	}

	// 1. Валюта стакана: 1 — доллары (кошелёк), 5 — рубли (аноним по IP).
	ob := get("https://steamcommunity.com/market/orderbook?q=Load&qp=%5B730%2C%22M4A4+%7C+Temukau+%28Field-Tested%29%22%5D")
	m := regexp.MustCompile(`"eCurrency":(\d+)`).FindSubmatch(ob)
	if m != nil {
		log.Printf("orderbook eCurrency = %s", m[1])
	}

	// 2. steamid в странице: 0 — аноним, иначе наш SteamID.
	page := get("https://steamcommunity.com/market/listings/730/G181020CC093004?l=english")
	s := regexp.MustCompile(`"steamid":"?(\d+)`).FindSubmatch(page)
	if s != nil {
		id := string(s[1])
		if id == "0" {
			log.Printf("page steamid = 0 (аноним)")
		} else {
			log.Printf("page steamid = …%s (вошли)", id[len(id)-4:]) // только последние цифры
		}
	}

	// 3. Валюта истории в странице.
	c := regexp.MustCompile(`ecurrency\\":(\d+)`).FindSubmatch(page)
	if c != nil {
		log.Printf("pricehistory ecurrency = %s", c[1])
	}
}

type result struct {
	status  int
	bytes   int64
	elapsed time.Duration
	err     error
}

func fetch(client *http.Client, u, cookie string) result {
	start := time.Now()

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return result{err: err}
	}
	if cookie != "" {
		req.Header.Set("Cookie", "steamLoginSecure="+cookie)
	}
	resp, err := client.Do(req)

	if err != nil {
		return result{err: err, elapsed: time.Since(start)}
	}
	defer resp.Body.Close()
	n, err := io.Copy(io.Discard, resp.Body) // Go сам распаковывает gzip, n — распакованный размер
	return result{status: resp.StatusCode, bytes: n, elapsed: time.Since(start), err: err}
}

func main() {
	endpoint := flag.String("endpoint", "page", "page | orderbook | overview")
	interval := flag.Duration("interval", 3*time.Second, "пауза между запросами")
	maxReq := flag.Int("max", 300, "максимум запросов до остановки")
	recheck := flag.Duration("recheck", 30*time.Second, "как часто проверять, снята ли блокировка")
	flag.Parse()

	log.SetFlags(log.Ltime)
	client := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Print("файл .env не найден, используем системное окружение")
	}
	login := os.Getenv("STEAM_LOGIN")
	password := os.Getenv("STEAM_PASSWORD")
	sharedSecret := os.Getenv("STEAM_SHARED_SECRET")

	cookie, err := auth.Login(ctx, client, login, password, sharedSecret)
	if err != nil {
		log.Fatalf("steam login: %v", err)
	}
	log.Println("вход выполнен")

	checkAuth(client, cookie)
	start := time.Now()
	ok := 0
	var blockedAt time.Time

	for i := 0; i < *maxReq; i++ {
		u, err := buildURL(*endpoint, itemsID[i%len(items)])
		if err != nil {
			log.Fatal(err)
		}
		r := fetch(client, u, cookie)
		switch {
		case r.err != nil:
			log.Printf("#%d error: %v", i+1, r.err)
		case r.status == http.StatusOK:
			ok++
			log.Printf("#%d 200  %7d KB  %v", i+1, r.bytes/1024, r.elapsed.Round(time.Millisecond))
		default:
			log.Printf("#%d %d  %v", i+1, r.status, r.elapsed.Round(time.Millisecond))
		}

		if r.status == http.StatusTooManyRequests {
			blockedAt = time.Now()
			break
		}
		time.Sleep(*interval)
	}

	took := time.Since(start)
	fmt.Println()
	log.Printf("endpoint=%s interval=%v", *endpoint, *interval)
	log.Printf("успешных запросов: %d за %v (%.1f в минуту)", ok, took.Round(time.Second), float64(ok)/took.Minutes())

	if blockedAt.IsZero() {
		log.Printf("429 не получили — лимит выше, попробуй меньший -interval")
		return
	}

	// Замер блокировки: пробуем раз в recheck, пока не пустят.
	log.Printf("получили 429, ждём снятия блокировки (проверка раз в %v)...", *recheck)
	for {
		time.Sleep(*recheck)
		u, _ := buildURL(*endpoint, items[0])
		r := fetch(client, u, cookie)
		log.Printf("проверка: %d (блокировка длится %v)", r.status, time.Since(blockedAt).Round(time.Second))
		if r.status == http.StatusOK {
			log.Printf("блокировка снята через %v", time.Since(blockedAt).Round(time.Second))
			return
		}
	}
}
