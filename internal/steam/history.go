package steam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type queryData struct {
	Queries []struct {
		QueryKey []json.RawMessage `json:"queryKey"`
		State    struct {
			Data json.RawMessage `json:"data"`
		} `json:"state"`
	} `json:"queries"`
}

type variantHistoryDTO struct {
	Currency int             `json:"ecurrency"`
	Prices   []pricePointDTO `json:"prices"`
}

type VariantHistory struct {
	Currency int
	Prices   []PricePoint
}

type pricePointDTO struct {
	Time        int64   `json:"time"`
	PriceMedian float64 `json:"price_median"`
	Purchases   int     `json:"purchases"`
}

type PricePoint struct {
	Time      time.Time
	Median    float64
	Purchases int
}

func extractSSR(html []byte) ([]byte, error) {
	idx := bytes.Index(html, []byte(`id="valve-ssr-data"`))
	if idx == -1 {
		return nil, fmt.Errorf("not found element in html")
	}
	sIdx := bytes.IndexByte(html[idx:], '>')
	if sIdx == -1 {
		return nil, fmt.Errorf("not found element in html")
	}
	firstPart := html[idx+sIdx+1:]
	tIdx := bytes.Index(firstPart, []byte(`</script>`))
	if tIdx == -1 {
		return nil, fmt.Errorf("not found element in html")
	}
	result := firstPart[:tIdx]
	return result, nil
}

type ssrAll struct {
	RenderContext struct {
		QueryData string `json:"queryData"`
	} `json:"renderContext"`
}

func extractQueryData(ssr []byte) (queryData, error) {
	var ssrAll ssrAll
	if err := json.Unmarshal(ssr, &ssrAll); err != nil {
		return queryData{}, fmt.Errorf("err extract ssr: %w", err)
	}

	if ssrAll.RenderContext.QueryData == "" {
		return queryData{}, fmt.Errorf("empty query data")
	}

	var qd queryData
	if err := json.Unmarshal([]byte(ssrAll.RenderContext.QueryData), &qd); err != nil {
		return queryData{}, fmt.Errorf("err extract query data: %w", err)
	}

	return qd, nil
}

func historyName(key []json.RawMessage) (string, bool) {
	if len(key) < 4 {
		return "", false
	}
	var scope, kind, name string
	if json.Unmarshal(key[0], &scope) != nil || scope != "market" {
		return "", false
	}
	if json.Unmarshal(key[1], &kind) != nil || kind != "pricehistory" {
		return "", false
	}
	if json.Unmarshal(key[3], &name) != nil || name == "" {
		return "", false
	}
	return name, true
}

func ParseGroupHistory(html []byte) (map[string]VariantHistory, error) {
	body, err := extractSSR(html)
	if err != nil {
		return nil, fmt.Errorf("parse group history extract ssr: %w", err)
	}
	queryData, err := extractQueryData(body)
	if err != nil {
		return nil, fmt.Errorf("parse group history extract query data: %w", err)
	}
	var result = make(map[string]VariantHistory)
	for _, query := range queryData.Queries {
		name, ok := historyName(query.QueryKey)
		if !ok {
			continue
		}

		var vh variantHistoryDTO

		if string(query.State.Data) == "null" || len(query.State.Data) == 0 {
			continue
		}

		if err := json.Unmarshal(query.State.Data, &vh); err != nil {
			return nil, fmt.Errorf("unmarshal variant history err: %w", err)
		}

		pricePoints := make([]PricePoint, 0, len(vh.Prices))
		for _, p := range vh.Prices {
			pricePoints = append(pricePoints, PricePoint{
				Time:      time.Unix(p.Time, 0).UTC(),
				Median:    p.PriceMedian,
				Purchases: p.Purchases,
			})
		}
		result[name] = VariantHistory{Currency: vh.Currency, Prices: pricePoints}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("empty result map")
	}
	return result, nil
}

// GroupPage загружает HTML страницы группы (все варианты скина).
func (c *Client) GroupPage(ctx context.Context, groupID string) ([]byte, error) {
	body, err := c.getBytes(ctx, c.pageLimiter, groupPageURL+groupID+"?l=english")
	if err != nil {
		return nil, fmt.Errorf("group page %s: %w", groupID, err)
	}
	return body, nil
}
