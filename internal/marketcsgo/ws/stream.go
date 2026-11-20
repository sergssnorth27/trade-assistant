package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/centrifugal/centrifuge-go"
	converter "github.com/sergssnorth27/trade-assistant/internal/domain"
)

const (
	getWSTokenUrl = "https://market.csgo.com/api/v2/get-ws-token?key="
	wsURL         = "wss://wsprice.csgo.com/connection/websocket"
	channel       = "public:items:730:usd"
)

type TokenResp struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
}

type Event struct {
	Event      string                `json:"event"`
	ID         json.Number           `json:"id"`
	ClassID    json.Number           `json:"class"`
	InstanceID json.Number           `json:"instance"`
	AssetID    json.Number           `json:"asset"`
	NameID     int                   `json:"name_id"`
	Price      converter.DollarPrice `json:"price"`
	ReceivedAt time.Time             `json:"-"` // время получения, ставим сами
}

func fetchToken(ctx context.Context, apiKey string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getWSTokenUrl+apiKey, nil)
	if err != nil {
		return "", fmt.Errorf("create ws token request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get ws token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode ws token resp, %w", err)
	}

	if !tokenResp.Success || tokenResp.Token == "" {
		return "", errors.New("market returned not token")
	}

	return tokenResp.Token, nil
}

func Run(ctx context.Context, apiKey string, out chan<- Event) error {
	clientWs := centrifuge.NewJsonClient(wsURL, centrifuge.Config{
		GetToken: func(cte centrifuge.ConnectionTokenEvent) (string, error) {
			return fetchToken(ctx, apiKey)
		},
	})
	defer clientWs.Close()

	// 3. Говорим, какой канал слушать, и что делать с каждым сообщением.
	sub, err := clientWs.NewSubscription(channel)
	if err != nil {
		log.Printf("new subscription: %v", err)
		return err
	}

	sub.OnPublication(func(e centrifuge.PublicationEvent) {

		var ev Event // новая переменная на каждое событие
		ev.ReceivedAt = time.Now()
		if err := json.Unmarshal(e.Data, &ev); err != nil {
			log.Printf("ws: bad message: %v: %s", err, e.Data)
			return
		}
		select {
		case out <- ev:
		default:
			log.Printf("ws: channel full, dropped event %s", ev.ID)
		}
	})

	if err := sub.Subscribe(); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	if err := clientWs.Connect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	log.Println("ws: connected")
	<-ctx.Done()
	return nil
}
