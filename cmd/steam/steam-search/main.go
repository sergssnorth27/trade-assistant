package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/sergssnorth27/trade-assistant/internal/steam"
	"github.com/sergssnorth27/trade-assistant/internal/steam/auth"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := godotenv.Load(); err != nil {
		log.Printf("load .env: %v", err)
	}
	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		log.Fatal("dsn is not set")
	}

	login := os.Getenv("STEAM_LOGIN")
	password := os.Getenv("STEAM_PASSWORD")
	sharedSecret := os.Getenv("STEAM_SHARED_SECRET")

	steamLoginSecure, err := auth.Login(ctx, &http.Client{}, login, password, sharedSecret)
	if err != nil {
		log.Fatal("fail auth in steam")
	}
	client, err := steam.NewAuthorizedClient(steamLoginSecure)
	if err != nil {
		log.Fatal("fail create auth user steam")
	}

	groupID := "G180120C2073004"
	html, err := client.GroupPage(ctx, groupID)
	if err != nil {
		log.Fatal("get group page err")
	}

	result, err := steam.ParseGroupHistory(html)
	if err != nil {
		log.Fatalf("err parse group histoy %v", err)
	}
	fmt.Println(result)

}
