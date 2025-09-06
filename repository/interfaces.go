package repository

import (
	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AccountRepository interface {
	Create(account *entity.Account) error
}

type WalletRepository interface {
	Create(tx *gorm.DB, wallet *entity.Wallet) error
	GetByAccountID(accountID uuid.UUID) ([]*entity.Wallet, error)
	GetByAccountAndAsset(tx *gorm.DB, accountID uuid.UUID, assetSymbol string) (*entity.Wallet, error)
	AddToBalance(tx *gorm.DB, accountID uuid.UUID, assetSymbol string, amount decimal.Decimal) error
	SubtractFromBalance(tx *gorm.DB, accountID uuid.UUID, assetSymbol string, amount decimal.Decimal) error
}

type OrderRepository interface {
	Create(tx *gorm.DB, order *entity.Order) error
	GetByID(id uuid.UUID, status ...string) (*entity.Order, error)
	GetOpenOrdersByInstrumentPair(instrumentPair string) ([]*entity.Order, error)
	UpdateStatus(id uuid.UUID, status string) error
	UpdateRemainingAndStatus(tx *gorm.DB, id uuid.UUID, quantity decimal.Decimal, status string) error
	GetMatchingOrders(
		tx *gorm.DB,
		accountID uuid.UUID,
		instrumentPair string,
		orderType string,
		price decimal.Decimal,
		isBuyOrder bool,
	) ([]*entity.Order, error)
}

type TradeRepository interface {
	Create(tx *gorm.DB, trade *entity.Trade) error
}


