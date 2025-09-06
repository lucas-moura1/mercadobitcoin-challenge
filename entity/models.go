package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Account struct {
	Base
	Name      string     `json:"name"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	Wallets   []*Wallet  `json:"wallets,omitempty" gorm:"foreignKey:AccountID"`
	Orders    []*Order   `json:"orders,omitempty" gorm:"foreignKey:AccountID"`
}

func (Account) TableName() string {
	return "account"
}

type Wallet struct {
	Base
	AccountID   uuid.UUID       `json:"account_id" gorm:"type:uuid"`
	AssetSymbol string          `json:"asset_symbol"`
	Balance     decimal.Decimal `json:"balance" gorm:"type:decimal(20,8)"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty"`
}

func (Wallet) TableName() string {
	return "wallet"
}

type Trade struct {
	ID            uuid.UUID       `json:"id" gorm:"type:uuid;primary_key"`
	BuyerOrderID  uuid.UUID       `json:"buyer_order_id" gorm:"type:uuid"`
	SellerOrderID uuid.UUID       `json:"seller_order_id" gorm:"type:uuid"`
	Price         decimal.Decimal `json:"price" gorm:"type:decimal(20,8)"`
	Quantity      decimal.Decimal `json:"quantity" gorm:"type:decimal(20,8)"`
	ExecutedAt    time.Time       `json:"executed_at"`
	DeletedAt     *time.Time      `json:"deleted_at,omitempty"`
}

func (Trade) TableName() string {
	return "trade"
}

func (t *Trade) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
