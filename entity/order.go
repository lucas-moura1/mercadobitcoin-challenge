package entity

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidPrice      = errors.New("price must be greater than zero")
	ErrInvalidQuantity   = errors.New("quantity must be greater than zero")
	ErrInvalidOrderType  = errors.New("invalid order type")
	ErrInvalidPairFormat = errors.New("invalid instrument pair format")
	ErrMaxQuantity       = errors.New("quantity exceeds maximum limit")
	ErrMaxPrice          = errors.New("price exceeds maximum limit")
)

type OrderType string

const (
	OrderTypeBuy  OrderType = "BUY"
	OrderTypeSell OrderType = "SELL"
)

type OrderStatus string

const (
	OrderStatusOpen      OrderStatus = "OPEN"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusPartial   OrderStatus = "PARTIALLY_FILLED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

const (
	MaxQuantity = 1000
	MaxPrice    = 100000000
)

type Order struct {
	Base
	AccountID         uuid.UUID       `json:"account_id" gorm:"type:uuid"`
	InstrumentPair    string          `json:"instrument_pair"`
	OrderType         string          `json:"order_type"`
	Price             decimal.Decimal `json:"price" gorm:"type:decimal(20,8)"`
	Quantity          decimal.Decimal `json:"quantity" gorm:"type:decimal(20,8)"`
	RemainingQuantity decimal.Decimal `json:"remaining_quantity" gorm:"type:decimal(20,8)"`
	Status            string          `json:"status"`
}

func (Order) TableName() string {
	return "order"
}

func (o *Order) Validate() error {
	if o.Price.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidPrice
	}

	if o.Quantity.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidQuantity
	}

	if o.Quantity.GreaterThan(decimal.NewFromInt(MaxQuantity)) {
		return ErrMaxQuantity
	}

	if o.Price.GreaterThan(decimal.NewFromInt(MaxPrice)) {
		return ErrMaxPrice
	}

	if o.OrderType != string(OrderTypeBuy) && o.OrderType != string(OrderTypeSell) {
		return ErrInvalidOrderType
	}

	if !IsValidInstrumentPair(o.InstrumentPair) {
		return ErrInvalidPairFormat
	}

	return nil
}

func IsValidInstrumentPair(pair string) bool {
	assets := strings.Split(pair, "_")
	return len(assets) == 2 && assets[0] != "" && assets[1] != ""
}

func (o *Order) GetRequiredAssetAndAmount() (string, decimal.Decimal) {
	assets := strings.Split(o.InstrumentPair, "_")

	if o.OrderType == string(OrderTypeBuy) {
		return assets[1], o.Price.Mul(o.Quantity)
	}

	return assets[0], o.Quantity
}
