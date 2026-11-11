package steam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

const groupPageURL = "https://steamcommunity.com/market/listings/730/"

var (
	ErrRateLimited = errors.New("steam: rate limited")
	ErrServer      = errors.New("steam: server error")
)

type Client struct {
	http          *http.Client
	searchLimiter *rate.Limiter
	pageLimiter   *rate.Limiter
}

func NewClient() *Client {
	return &Client{
		http:          &http.Client{Timeout: 30 * time.Second},
		searchLimiter: rate.NewLimiter(rate.Every(time.Minute/16), 5),
		pageLimiter:   rate.NewLimiter(rate.Every(time.Minute/16), 5),
	}
}

// NewAuthorizedClient создаёт клиент, который ходит в Steam от имени аккаунта.
// steamLoginSecure — значение cookie вида "<steamid>%7C%7C<access_token>".
func NewAuthorizedClient(steamLoginSecure string) (*Client, error) {
	if steamLoginSecure == "" {
		return nil, errors.New("steam: empty steamLoginSecure")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	u, _ := url.Parse("https://steamcommunity.com")
	jar.SetCookies(u, []*http.Cookie{{
		Name:     "steamLoginSecure",
		Value:    steamLoginSecure,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	}})

	c := NewClient()
	c.http.Jar = jar
	return c, nil

}

// getJSON загружает ответ и разбирает его как JSON в dst (dst — указатель).
func (c *Client) getJSON(ctx context.Context, limiter *rate.Limiter, rawURL string, dst any) error {
	body, err := c.getBytes(ctx, limiter, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

// getBytes загружает тело ответа. Повторяет запрос только при ошибках сервера (5xx).
func (c *Client) getBytes(ctx context.Context, limiter *rate.Limiter, rawURL string) ([]byte, error) {
	const attempts = 3

	var (
		body []byte
		err  error
	)
	for i := 1; i <= attempts; i++ {
		body, err = c.loadOnce(ctx, limiter, rawURL) // "=", а не ":=": пишем во внешние переменные
		if err == nil {
			return body, nil
		}
		if !errors.Is(err, ErrServer) {
			return nil, err // 429, отмена контекста и т. п. — повторять бессмысленно
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", attempts, err)
}

// loadOnce — одна попытка: дождаться ограничителя, сделать запрос, проверить статус, прочитать тело.
func (c *Client) loadOnce(ctx context.Context, limiter *rate.Limiter, rawURL string) ([]byte, error) {
	if err := limiter.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}
		return body, nil
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("url %s: %w", rawURL, ErrRateLimited)
	case resp.StatusCode >= http.StatusInternalServerError:
		return nil, fmt.Errorf("status %d: %w", resp.StatusCode, ErrServer)
	default:
		return nil, fmt.Errorf("unexpected status %d, url: %s", resp.StatusCode, rawURL)
	}
}
