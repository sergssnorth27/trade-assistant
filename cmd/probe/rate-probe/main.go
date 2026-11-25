// go run ./cmd/rate-probe
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

func main() {
	groupID := "G181020CC093004" // M4A4 | Temukau
	if len(os.Args) > 1 {
		groupID = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := steam.NewClient() // без логина
	html, err := client.GroupPage(ctx, groupID)

	switch {
	case err == nil:
		fmt.Printf("200 OK: страница получена, %d байт → IP не ограничен, ограничен аккаунт бота\n", len(html))
	case errors.Is(err, steam.ErrRateLimited):
		fmt.Printf("429: %v\n", err)
	default:
		fmt.Printf("другая ошибка: %v\n", err)
	}
}
