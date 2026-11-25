package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/sergssnorth27/trade-assistant/internal/steam/auth"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Print("файл .env не найден, используем системное окружение")
	}

	login := os.Getenv("STEAM_LOGIN")
	password := os.Getenv("STEAM_PASSWORD")
	sharedSecret := os.Getenv("STEAM_SHARED_SECRET")

	switch {
	case login == "":
		log.Print("STEAM_LOGIN must be set in .env")
	case password == "":
		log.Print("STEAM_PASSWORD must be set in .env")
	case sharedSecret == "":
		log.Print("sharedSecret must be set in .env")
	}

	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	sess, err := auth.NewSession(client)
	if err != nil {
		log.Print("не удалось создать сессию")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// шаг 1 — получаем sessionid из cookie jar
	_, cookies, err := sess.GetSessionId(ctx)
	if err != nil {
		log.Fatal("get session id: ", err)
	}

	var sessionID string
	for _, c := range cookies {
		if c.Name == "sessionid" {
			sessionID = c.Value
		}
	}
	fmt.Printf("sessionID - %s\n", sessionID)

	// шаг 2 — получаем RSA публичный ключ для шифрования пароля
	body, _, err := sess.GetPasswordRSAPublicKey(ctx, login)
	if err != nil {
		log.Fatal("get RSA public key: ", err)
	}

	rsaResp, err := auth.DeserializeCAuthentication_GetPasswordRSAPublicKey_Response(
		base64.StdEncoding.EncodeToString(body),
	)
	if err != nil {
		log.Fatal("deserialize RSA response: ", err)
	}

	// шаг 3 — шифруем пароль
	pub, err := auth.BuildRSAPublicKey(rsaResp.GetPublickeyMod(), rsaResp.GetPublickeyExp())
	if err != nil {
		log.Fatal("build RSA key: ", err)
	}

	encryptedPassword, err := auth.EncryptPKCS1v15Base64(pub, password, false)
	if err != nil {
		log.Fatal("encrypt password: ", err)
	}
	fmt.Printf("зашифрованный пароль - %s\n", encryptedPassword)

	// шаг 4 — начинаем сессию авторизации
	body, _, err = sess.BeginAuthSessionViaCredentials(ctx, login, encryptedPassword, rsaResp.GetTimestamp())
	if err != nil {
		log.Fatal("begin auth session: ", err)
	}

	beginResp, err := auth.DeserializeCAuthentication_BeginAuthSessionViaCredentials_Response(
		base64.StdEncoding.EncodeToString(body),
	)
	if err != nil {
		log.Fatal("deserialize begin auth response: ", err)
	}

	// шаг 5 — генерируем Steam Guard код и подтверждаем
	code, err := auth.GenerateSteamGuardCode(sharedSecret, time.Now().Unix())
	if err != nil {
		log.Fatal("generate steam guard code: ", err)
	}
	fmt.Printf("steam guard code - %s\n", code)

	_, _, err = sess.UpdateAuthSessionWithSteamGuardCode(
		ctx,
		beginResp.GetClientId(),
		beginResp.GetSteamid(),
		code,
	)
	if err != nil {
		log.Fatal("update auth session with steam guard: ", err)
	}

	// шаг 6 — ждём завершения аутентификации и получаем refresh token
	body, _, err = sess.PollAuthSessionStatus(ctx, beginResp.GetClientId(), beginResp.GetRequestId(), 0)
	if err != nil {
		log.Fatal("poll auth session status: ", err)
	}

	pollResp, err := auth.DeserializePollAuthSessionStatus_Response(
		base64.StdEncoding.EncodeToString(body),
	)
	if err != nil {
		log.Fatal("deserialize poll response: ", err)
	}

	// шаг 7 — финализируем логин, получаем steamRefresh_steam cookie
	_, cookies, err = sess.FinalizeLogin(ctx, pollResp.GetRefreshToken(), sessionID)
	if err != nil {
		log.Fatal("finalize login: ", err)
	}

	var steamRefresh string
	for _, c := range cookies {
		if c.Name == "steamRefresh_steam" {
			steamRefresh = c.Value
		}
	}

	if steamRefresh == "" {
		log.Fatal("steamRefresh_steam cookie not found — finalize login failed")
	}

	fmt.Printf("steam refresh token - %s\n", steamRefresh)

	// шаг 8 — проверяем что сессия действительно работает
	// steamLoginSecure = steamid||access_token (|| закодировано как %7C%7C, как в браузере)
	accessToken := pollResp.GetAccessToken()
	if accessToken == "" {
		log.Fatal("access token is empty — poll auth session status returned no token")
	}
	fmt.Printf("steam access token - %s\n", accessToken)
	steamLoginSecure := fmt.Sprintf("%d%%7C%%7C%s", beginResp.GetSteamid(), accessToken)

	body, _, err = sess.SetCookiesAndAuthentication(ctx, steamLoginSecure)
	if err != nil {
		log.Fatal("set cookies and check auth: ", err)
	}

	ok, err := auth.IsAuthenticated(body)
	if err != nil {
		log.Fatal("parse auth check html: ", err)
	}

	if ok {
		fmt.Println("✓ authenticated successfully")
	} else {
		fmt.Println("✗ not authenticated")
	}
}
