package usecase

import (
	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type OrderUseCase interface {
	CreateOrder(order *entity.Order) error
	CancelOrder(id uuid.UUID) error
	GetOrderBook(instrumentPair string) (*OrderBook, error)
}

type AccountUseCase interface {
	GetAccountBalance(accountID uuid.UUID) ([]*entity.Wallet, error)
}

type OrderBook struct {
	InstrumentPair string
	Bids           []*OrderBookEntry
	Asks           []*OrderBookEntry
}

type OrderBookEntry struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

type TradeExecutor interface {
	Execute(tx *gorm.DB, order, matchingOrder *entity.Order, qty decimal.Decimal) error
}
