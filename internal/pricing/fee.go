package pricing

import "github.com/sergssnorth27/trade-assistant/internal/domain"

// Комиссии Steam Market для CS2, в процентах от суммы, которую получает продавец.
const (
	steamFeePercent     = 5  // комиссия Steam
	publisherFeePercent = 10 // комиссия Valve за CS2
)

// BuyerPays возвращает, сколько заплатит покупатель, если продавец хочет получить sellerCents.
// Каждая комиссия округляется вниз, но не бывает меньше 1 цента.
// Пример: продавец 5219 → 5219 + 260 + 521 = 6000.
func BuyerPays(sellerCents int64) int64 {
	if sellerCents <= 0 {
		return 0
	}
	steamFee := max(sellerCents*steamFeePercent/100, 1)
	publisherFee := max(sellerCents*publisherFeePercent/100, 1)
	return sellerCents + steamFee + publisherFee
}

// SellerReceives возвращает, сколько получит продавец, если покупатель платит buyerPrice.
// Обратной формулы у комиссии Steam нет (округление и минимум в 1 цент), поэтому ищем
// наибольшую сумму продавца, при которой покупатель платит не больше buyerPrice.
func SellerReceives(buyerPrice domain.Price) domain.Price {
	cents := int64(buyerPrice) / 10 // тысячные доли → центы; Steam работает в центах
	if cents < 3 {
		return 0 // минимальный лот: продавцу 1 цент, покупателю 1 + 1 + 1 = 3
	}

	seller := cents * 100 / 115 // первое приближение: комиссия около 15%
	for BuyerPays(seller+1) <= cents {
		seller++
	}
	for seller > 0 && BuyerPays(seller) > cents {
		seller--
	}
	return domain.Price(seller * 10)
}
