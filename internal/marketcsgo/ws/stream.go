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
	"github.com/sergssnorth27/trade-assistant/internal/marketcsgo/converter"
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
	ID         int64                 `json:"id,string"`
	ClassID    uint64                `json:"class,string"`
	InstanceID uint64                `json:"instance,string"`
	AssetID    uint64                `json:"asset,string"`
	NameID     int                   `json:"name_id"`
	Price      converter.DollarPrice `json:"price"`
}

func fetchToken(ctx context.Context, apiKey string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := http.Get(getWSTokenUrl + apiKey)
	if err != nil {
		return "", fmt.Errorf("get ws token %w", err)
	}

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
		if err := json.Unmarshal(e.Data, &ev); err != nil {
			log.Printf("ws: bad message: %v: %s", err, e.Data)
			return
		}
		select {
		case out <- ev:
		default:
			log.Printf("ws: channel full, dropped event %d", ev.ID)
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
