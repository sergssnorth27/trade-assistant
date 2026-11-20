package evaluator

import (
	"context"
	"log"
	"time"

	"github.com/sergssnorth27/trade-assistant/internal/domain"
)

// Hit — сигнал: на market.csgo появился лот дешевле порога покупки.
type Hit struct {
	Name      string
	LotPrice  domain.Price
	Threshold Threshold
	At        time.Time
}

// hitCooldown — один и тот же предмет пишем не чаще раза в это время:
// снапшот приходит каждые несколько минут, и без этого один лот попадал бы в лог десятки раз.
const hitCooldown = time.Hour

// LogHits читает сигналы и пишет их в лог. Позже здесь же будет запись в paper_trades.
func LogHits(ctx context.Context, hits <-chan Hit) {
	lastSeen := make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			return
		case h := <-hits:
			if last, ok := lastSeen[h.Name]; ok && h.At.Sub(last) < hitCooldown {
				continue // уже писали этот предмет недавно
			}
			lastSeen[h.Name] = h.At

			roi := float64(h.Threshold.Receive)/float64(h.LotPrice) - 1
			log.Printf("hit: %s | lot %s | max buy %s | sell %s | receive %s | roi %.1f%%",
				h.Name, h.LotPrice, h.Threshold.MaxBuy, h.Threshold.SellPrice, h.Threshold.Receive, roi*100)
		}
	}
}
