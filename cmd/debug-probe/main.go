// debug-probe — один GET страницы группы из Go с подробным выводом ответа.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const pageURL = "https://steamcommunity.com/market/listings/730/G181020CC093004?l=english"

func main() {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	fmt.Println("status:  ", resp.Status)
	fmt.Println("protocol:", resp.Proto)        // HTTP/2.0 или HTTP/1.1
	fmt.Println("final URL:", resp.Request.URL) // если был редирект — куда привёл
	fmt.Println("--- headers ---")
	for k, v := range resp.Header {
		if k == "Set-Cookie" {
			continue // cookie не печатаем
		}
		fmt.Printf("%s: %v\n", k, v)
	}
	fmt.Println("--- body (первые 500 байт) ---")
	if len(body) > 500 {
		body = body[:500]
	}
	fmt.Println(string(body))
}
