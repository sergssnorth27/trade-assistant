package auth

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"

	"golang.org/x/net/publicsuffix"
)

type Session struct {
	client  *http.Client
	BaseURL string
	Origin  string
}

func NewSession(client *http.Client) (*Session, error) {
	if client == nil {
		return nil, fmt.Errorf("auth: nil http client")
	}
	if client.Jar == nil {
		jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		if err != nil {
			return nil, fmt.Errorf("auth: create cookie jar: %w", err)
		}
		client.Jar = jar
	}

	return &Session{
		client:  client,
		BaseURL: "https://api.steampowered.com",
		Origin:  "https://steamcommunity.com",
	}, nil
}

func (s *Session) Do(ctx context.Context, req *http.Request) ([]byte, []*http.Cookie, error) {
	req = req.WithContext(ctx)

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Cookies(), fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return body, resp.Cookies(), nil
}

func (s *Session) UpdateAuthSessionWithSteamGuardCode(
	ctx context.Context,
	сlientId, steamId uint64,
	code string,
) ([]byte, []*http.Cookie, error) {

	inputB64, err := SerializeCAuthentication_UpdateAuthSessionWithSteamGuardCode_Request(
		сlientId,
		steamId,
		code,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("serialize request: %w", err)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("input_protobuf_encoded", inputB64); err != nil {
		return nil, nil, fmt.Errorf("write field: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, nil, fmt.Errorf("writer close: %w", err)
	}

	u := s.BaseURL + "/IAuthenticationService/UpdateAuthSessionWithSteamGuardCode/v1"
	req, err := http.NewRequestWithContext(ctx, "POST", u, &buf)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", s.Origin)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	body, cookies, err := s.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	return body, cookies, nil
}

func (s *Session) PollAuthSessionStatus(
	ctx context.Context,
	сlientId uint64, requestId []byte,
	tokenToRevoke uint64,
) ([]byte, []*http.Cookie, error) {

	inputB64, err := SerializePollAuthSessionStatus_Request(
		сlientId,
		requestId,
		tokenToRevoke,
	)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("input_protobuf_encoded", inputB64); err != nil {
		return nil, nil, fmt.Errorf("write field: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, nil, fmt.Errorf("writer close: %w", err)
	}

	u := s.BaseURL + "/IAuthenticationService/PollAuthSessionStatus/v1"
	req, err := http.NewRequestWithContext(ctx, "POST", u, &buf)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Origin", s.Origin)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	body, cookies, err := s.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	return body, cookies, nil
}
