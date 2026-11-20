package steam

type OrderBookResponse struct {
	Data struct {
		Success bool `json:"success"`
		Data    struct {
			AmtMaxBuyOrder  int64 `json:"amtMaxBuyOrder"`
			AmtMinSellOrder int64 `json:"amtMinSellOrder"`
			ECurrency       int64 `json:"eCurrency"`
			CBuyOrders      int64 `json:"cBuyOrders"`
			CSellOrders     int64 `json:"cSellOrders"`
		} `json:"data"`
	} `json:"data"`
}
