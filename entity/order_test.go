package entity

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestOrderValidate(t *testing.T) {
	tests := []struct {
		name    string
		order   Order
		wantErr bool
		errIs   error
	}{
		{
			name: "valid BUY",
			order: Order{
				InstrumentPair: "BTC_BRL",
				OrderType:      string(OrderTypeBuy),
				Price:          decimal.RequireFromString("100.00"),
				Quantity:       decimal.RequireFromString("1.0"),
			},
		},
		{
			name: "valid SELL",
			order: Order{
				InstrumentPair: "ETH_BTC",
				OrderType:      string(OrderTypeSell),
				Price:          decimal.RequireFromString("0.060"),
				Quantity:       decimal.RequireFromString("2.5"),
			},
		},
		{
			name: "invalid price zero",
			order: Order{
				InstrumentPair: "BTC_BRL",
				OrderType:      string(OrderTypeBuy),
				Price:          decimal.Zero,
				Quantity:       decimal.RequireFromString("1"),
			},
			wantErr: true,
			errIs:   ErrInvalidPrice,
		},
		{
			name: "invalid price negative",
			order: Order{
				InstrumentPair: "BTC_BRL",
				OrderType:      string(OrderTypeSell),
				Price:          decimal.RequireFromString("-1"),
				Quantity:       decimal.RequireFromString("1"),
			},
			wantErr: true,
			errIs:   ErrInvalidPrice,
		},
		{
			name: "invalid quantity zero",
			order: Order{
				InstrumentPair: "BTC_BRL",
				OrderType:      string(OrderTypeBuy),
				Price:          decimal.RequireFromString("100"),
				Quantity:       decimal.Zero,
			},
			wantErr: true,
			errIs:   ErrInvalidQuantity,
		},
		{
			name: "invalid quantity negative",
			order: Order{
				InstrumentPair: "BTC_BRL",
				OrderType:      string(OrderTypeBuy),
				Price:          decimal.RequireFromString("100"),
				Quantity:       decimal.RequireFromString("-0.1"),
			},
			wantErr: true,
			errIs:   ErrInvalidQuantity,
		},
		{
			name: "exceeds max price",
			order: Order{
				InstrumentPair: "BTC_BRL",
				OrderType:      string(OrderTypeBuy),
				Price:          decimal.NewFromInt(MaxPrice + 1),
				Quantity:       decimal.RequireFromString("1"),
			},
			wantErr: true,
			errIs:   ErrMaxPrice,
		},
		{
			name: "exceeds max quantity",
			order: Order{
				InstrumentPair: "BTC_BRL",
				OrderType:      string(OrderTypeSell),
				Price:          decimal.RequireFromString("100"),
				Quantity:       decimal.NewFromInt(MaxQuantity + 1),
			},
			wantErr: true,
			errIs:   ErrMaxQuantity,
		},
		{
			name: "invalid order type",
			order: Order{
				InstrumentPair: "BTC_BRL",
				OrderType:      "HOLD",
				Price:          decimal.RequireFromString("100"),
				Quantity:       decimal.RequireFromString("1"),
			},
			wantErr: true,
			errIs:   ErrInvalidOrderType,
		},
		{
			name: "invalid pair missing underscore",
			order: Order{
				InstrumentPair: "BTCBRL",
				OrderType:      string(OrderTypeBuy),
				Price:          decimal.RequireFromString("100"),
				Quantity:       decimal.RequireFromString("1"),
			},
			wantErr: true,
			errIs:   ErrInvalidPairFormat,
		},
		{
			name: "invalid pair empty base",
			order: Order{
				InstrumentPair: "_BRL",
				OrderType:      string(OrderTypeBuy),
				Price:          decimal.RequireFromString("100"),
				Quantity:       decimal.RequireFromString("1"),
			},
			wantErr: true,
			errIs:   ErrInvalidPairFormat,
		},
		{
			name: "invalid pair empty quote",
			order: Order{
				InstrumentPair: "BTC_",
				OrderType:      string(OrderTypeBuy),
				Price:          decimal.RequireFromString("100"),
				Quantity:       decimal.RequireFromString("1"),
			},
			wantErr: true,
			errIs:   ErrInvalidPairFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestIsValidInstrumentPair(t *testing.T) {
	tests := []struct {
		pair string
		want bool
	}{
		{"BTC_BRL", true},
		{"ETH_BTC", true},
		{"BTCBRL", false},
		{"BTC_", false},
		{"_BRL", false},
		{"", false},
		{"ONE_TWO_THREE", false},
	}

	for _, tc := range tests {
		t.Run(tc.pair, func(t *testing.T) {
			got := IsValidInstrumentPair(tc.pair)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetRequiredAssetAndAmount(t *testing.T) {
	tests := []struct {
		name          string
		orderType     string
		pair          string
		price         string
		qty           string
		wantAsset     string
		wantAmountStr string
	}{
		{
			name:          "BUY returns quote and total",
			orderType:     string(OrderTypeBuy),
			pair:          "BTC_BRL",
			price:         "200000.00",
			qty:           "0.5",
			wantAsset:     "BRL",
			wantAmountStr: "100000.00",
		},
		{
			name:          "SELL returns base and quantity",
			orderType:     string(OrderTypeSell),
			pair:          "BTC_BRL",
			price:         "200000.00",
			qty:           "0.5",
			wantAsset:     "BTC",
			wantAmountStr: "0.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := Order{
				InstrumentPair: tt.pair,
				OrderType:      tt.orderType,
				Price:          decimal.RequireFromString(tt.price),
				Quantity:       decimal.RequireFromString(tt.qty),
			}
			asset, amount := o.GetRequiredAssetAndAmount()

			assert.Equal(t, tt.wantAsset, asset)

			wantAmount := decimal.RequireFromString(tt.wantAmountStr)
			assert.Truef(t, amount.Equal(wantAmount), "amount = %s, want %s", amount.String(), wantAmount.String())
		})
	}
}
