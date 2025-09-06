package usecase

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/lucas-moura1/mercadobitcoin-challenge/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newInMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}
	return db
}

func TestOrderUseCase_CancelOrder(t *testing.T) {
	orderID := uuid.New()

	tests := []struct {
		name        string
		setupMock   func(or *repository.MockOrderRepository)
		wantErr     bool
		wantNilResp bool
	}{
		{
			name: "success - cancels open order",
			setupMock: func(or *repository.MockOrderRepository) {
				or.EXPECT().
					GetByID(orderID, string(entity.OrderStatusOpen)).
					Return(&entity.Order{
						Base:   entity.Base{ID: orderID},
						Status: string(entity.OrderStatusOpen),
					}, nil).
					Times(1)

				or.EXPECT().
					UpdateStatus(orderID, string(entity.OrderStatusCancelled)).
					Return(nil).
					Times(1)
			},
			wantErr:     false,
			wantNilResp: false,
		},
		{
			name: "no-op - order not found",
			setupMock: func(or *repository.MockOrderRepository) {
				or.EXPECT().
					GetByID(orderID, string(entity.OrderStatusOpen)).
					Return(nil, nil).
					Times(1)
			},
			wantErr:     false,
			wantNilResp: true,
		},
		{
			name: "error - GetByID fails",
			setupMock: func(or *repository.MockOrderRepository) {
				or.EXPECT().
					GetByID(orderID, string(entity.OrderStatusOpen)).
					Return(nil, errors.New("db error")).
					Times(1)
			},
			wantErr:     true,
			wantNilResp: true,
		},
		{
			name: "error - UpdateStatus fails",
			setupMock: func(or *repository.MockOrderRepository) {
				or.EXPECT().
					GetByID(orderID, string(entity.OrderStatusOpen)).
					Return(&entity.Order{
						Base:   entity.Base{ID: orderID},
						Status: string(entity.OrderStatusOpen),
					}, nil).
					Times(1)

				or.EXPECT().
					UpdateStatus(orderID, string(entity.OrderStatusCancelled)).
					Return(errors.New("update failed")).
					Times(1)
			},
			wantErr:     true,
			wantNilResp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			orderRepo := repository.NewMockOrderRepository(ctrl)
			walletRepo := repository.NewMockWalletRepository(ctrl)
			tradeRepo := repository.NewMockTradeRepository(ctrl)

			tt.setupMock(orderRepo)
			uc := NewOrderUseCase(
				zap.NewNop().Sugar(),
				orderRepo,
				walletRepo,
				tradeRepo,
				nil,
			)

			err := uc.CancelOrder(orderID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}

func TestOrderUseCase_GetOrderBook(t *testing.T) {
	tests := []struct {
		name           string
		instrumentPair string
		mockSetup      func(or *repository.MockOrderRepository)
		wantErr        bool
		errIs          error
		wantNilResp    bool
	}{
		{
			name:           "aggregates and sorts bids/asks",
			instrumentPair: "BTC_BRL",
			mockSetup: func(or *repository.MockOrderRepository) {
				orders := []*entity.Order{
					{OrderType: string(entity.OrderTypeBuy), Price: decimal.RequireFromString("100"), RemainingQuantity: decimal.RequireFromString("1.0")},
					{OrderType: string(entity.OrderTypeBuy), Price: decimal.RequireFromString("99"), RemainingQuantity: decimal.RequireFromString("2.0")},
					{OrderType: string(entity.OrderTypeBuy), Price: decimal.RequireFromString("100"), RemainingQuantity: decimal.RequireFromString("0.4")},

					{OrderType: string(entity.OrderTypeSell), Price: decimal.RequireFromString("101"), RemainingQuantity: decimal.RequireFromString("0.5")},
					{OrderType: string(entity.OrderTypeSell), Price: decimal.RequireFromString("103"), RemainingQuantity: decimal.RequireFromString("0.2")},
					{OrderType: string(entity.OrderTypeSell), Price: decimal.RequireFromString("101"), RemainingQuantity: decimal.RequireFromString("0.3")},
				}
				or.EXPECT().
					GetOpenOrdersByInstrumentPair("BTC_BRL").
					Return(orders, nil).
					Times(1)
			},
			wantErr:     false,
			wantNilResp: false,
		},
		{
			name:           "invalid instrument pair",
			instrumentPair: "BTCBRL",
			mockSetup:      func(or *repository.MockOrderRepository) {},
			wantErr:        true,
			errIs:          entity.ErrInvalidPairFormat,
			wantNilResp:    true,
		},
		{
			name:           "repository error",
			instrumentPair: "BTC_BRL",
			mockSetup: func(or *repository.MockOrderRepository) {
				or.EXPECT().
					GetOpenOrdersByInstrumentPair("BTC_BRL").
					Return(nil, errors.New("db error")).
					Times(1)
			},
			wantErr:     true,
			wantNilResp: true,
		},
		{
			name:           "no open orders",
			instrumentPair: "BTC_BRL",
			mockSetup: func(or *repository.MockOrderRepository) {
				or.EXPECT().
					GetOpenOrdersByInstrumentPair("BTC_BRL").
					Return(nil, nil).
					Times(1)
			},
			wantErr:     false,
			errIs:       nil,
			wantNilResp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			orderRepo := repository.NewMockOrderRepository(ctrl)
			walletRepo := repository.NewMockWalletRepository(ctrl)
			tradeRepo := repository.NewMockTradeRepository(ctrl)

			tt.mockSetup(orderRepo)

			uc := NewOrderUseCase(zap.NewNop().Sugar(), orderRepo, walletRepo, tradeRepo, nil)

			ob, err := uc.GetOrderBook(tt.instrumentPair)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
				return
			}

			if tt.wantNilResp {
				assert.Nil(t, ob)
				return
			}

			assert.Nil(t, err)
			assert.NotNil(t, ob)
			assert.Equal(t, "BTC_BRL", ob.InstrumentPair)

			if assert.Len(t, ob.Bids, 2) {
				assert.Equal(t, "100", ob.Bids[0].Price.String())
				assert.Equal(t, "1.4", ob.Bids[0].Quantity.String()) // 1.0 + 0.4
				assert.Equal(t, "99", ob.Bids[1].Price.String())
				assert.Equal(t, "2", ob.Bids[1].Quantity.String())
			}

			if assert.Len(t, ob.Asks, 2) {
				assert.Equal(t, "101", ob.Asks[0].Price.String())
				assert.Equal(t, "0.8", ob.Asks[0].Quantity.String()) // 0.5 + 0.3
				assert.Equal(t, "103", ob.Asks[1].Price.String())
				assert.Equal(t, "0.2", ob.Asks[1].Quantity.String())
			}
		})
	}
}

func TestOrderUseCase_CreateOrder(t *testing.T) {
	accountID := uuid.New()
	validBuy := &entity.Order{
		AccountID:      accountID,
		InstrumentPair: "BTC_BRL",
		OrderType:      string(entity.OrderTypeBuy),
		Price:          decimal.RequireFromString("200000.00"),
		Quantity:       decimal.RequireFromString("0.50"),
	}
	validSell := &entity.Order{
		AccountID:      accountID,
		InstrumentPair: "BTC_BRL",
		OrderType:      string(entity.OrderTypeSell),
		Price:          decimal.RequireFromString("200000.00"),
		Quantity:       decimal.RequireFromString("0.50"),
	}
	invalidOrder := &entity.Order{
		AccountID:      accountID,
		InstrumentPair: "BTC_BRL",
		OrderType:      string(entity.OrderTypeBuy),
		Price:          decimal.Zero,
		Quantity:       decimal.RequireFromString("0.50"),
	}

	type args struct {
		order *entity.Order
	}
	tests := []struct {
		name      string
		args      args
		mockSetup func(
			or *repository.MockOrderRepository,
			wr *repository.MockWalletRepository,
			tr *repository.MockTradeRepository,
			o *entity.Order,
		)
		wantErr bool
	}{
		{
			name: "success - BUY without matches",
			args: args{order: validBuyClone(validBuy)},
			mockSetup: func(
				or *repository.MockOrderRepository,
				wr *repository.MockWalletRepository,
				tr *repository.MockTradeRepository,
				o *entity.Order,
			) {
				required := o.Price.Mul(o.Quantity)
				wr.EXPECT().
					GetByAccountAndAsset(gomock.Any(), o.AccountID, "BRL").
					Return(&entity.Wallet{AccountID: o.AccountID, AssetSymbol: "BRL", Balance: required}, nil).
					Times(1)

				or.EXPECT().
					Create(gomock.Any(), o).
					Return(nil).
					Times(1)

				or.EXPECT().
					GetMatchingOrders(gomock.Any(), o.AccountID, o.InstrumentPair, "SELL", o.Price, true).
					Return([]*entity.Order{}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "success - SELL without matches (checks base asset coverage)",
			args: args{order: validSellClone(validSell)},
			mockSetup: func(
				or *repository.MockOrderRepository,
				wr *repository.MockWalletRepository,
				tr *repository.MockTradeRepository,
				o *entity.Order,
			) {
				wr.EXPECT().
					GetByAccountAndAsset(gomock.Any(), o.AccountID, "BTC").
					Return(&entity.Wallet{AccountID: o.AccountID, AssetSymbol: "BTC", Balance: o.Quantity}, nil).
					Times(1)

				or.EXPECT().
					Create(gomock.Any(), o).
					Return(nil).
					Times(1)

				or.EXPECT().
					GetMatchingOrders(gomock.Any(), o.AccountID, o.InstrumentPair, "BUY", o.Price, false).
					Return([]*entity.Order{}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "validation error - invalid order",
			args: args{order: invalidOrder},
			mockSetup: func(
				or *repository.MockOrderRepository,
				wr *repository.MockWalletRepository,
				tr *repository.MockTradeRepository,
				o *entity.Order,
			) {
			},
			wantErr: true,
		},
		{
			name: "error - wallet not found",
			args: args{order: validBuyClone(validBuy)},
			mockSetup: func(
				or *repository.MockOrderRepository,
				wr *repository.MockWalletRepository,
				tr *repository.MockTradeRepository,
				o *entity.Order,
			) {
				wr.EXPECT().
					GetByAccountAndAsset(gomock.Any(), o.AccountID, "BRL").
					Return(nil, nil).
					Times(1)
			},
			wantErr: true,
		},
		{
			name: "error - insufficient balance (BUY)",
			args: args{order: validBuyClone(validBuy)},
			mockSetup: func(
				or *repository.MockOrderRepository,
				wr *repository.MockWalletRepository,
				tr *repository.MockTradeRepository,
				o *entity.Order,
			) {
				needed := o.Price.Mul(o.Quantity)
				insufficient := needed.Sub(decimal.RequireFromString("1"))
				wr.EXPECT().
					GetByAccountAndAsset(gomock.Any(), o.AccountID, "BRL").
					Return(&entity.Wallet{AccountID: o.AccountID, AssetSymbol: "BRL", Balance: insufficient}, nil).
					Times(1)
			},
			wantErr: true,
		},
		{
			name: "error - repository Create fails",
			args: args{order: validBuyClone(validBuy)},
			mockSetup: func(
				or *repository.MockOrderRepository,
				wr *repository.MockWalletRepository,
				tr *repository.MockTradeRepository,
				o *entity.Order,
			) {
				required := o.Price.Mul(o.Quantity)
				wr.EXPECT().
					GetByAccountAndAsset(gomock.Any(), o.AccountID, "BRL").
					Return(&entity.Wallet{AccountID: o.AccountID, AssetSymbol: "BRL", Balance: required}, nil).
					Times(1)

				or.EXPECT().
					Create(gomock.Any(), o).
					Return(assert.AnError).
					Times(1)
			},
			wantErr: true,
		},
		{
			name: "error - GetMatchingOrders fails",
			args: args{order: validBuyClone(validBuy)},
			mockSetup: func(
				or *repository.MockOrderRepository,
				wr *repository.MockWalletRepository,
				tr *repository.MockTradeRepository,
				o *entity.Order,
			) {
				required := o.Price.Mul(o.Quantity)
				wr.EXPECT().
					GetByAccountAndAsset(gomock.Any(), o.AccountID, "BRL").
					Return(&entity.Wallet{AccountID: o.AccountID, AssetSymbol: "BRL", Balance: required}, nil).
					Times(1)

				or.EXPECT().
					Create(gomock.Any(), o).
					Return(nil).
					Times(1)

				or.EXPECT().
					GetMatchingOrders(gomock.Any(), o.AccountID, o.InstrumentPair, "SELL", o.Price, true).
					Return(nil, assert.AnError).
					Times(1)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			db := newInMemoryDB(t)

			orderRepo := repository.NewMockOrderRepository(ctrl)
			walletRepo := repository.NewMockWalletRepository(ctrl)
			tradeRepo := repository.NewMockTradeRepository(ctrl)

			tt.mockSetup(orderRepo, walletRepo, tradeRepo, tt.args.order)

			uc := NewOrderUseCase(zap.NewNop().Sugar(), orderRepo, walletRepo, tradeRepo, db)
			err := uc.CreateOrder(tt.args.order)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.Nil(t, err)

			assert.Equal(t, string(entity.OrderStatusOpen), tt.args.order.Status)
			assert.Equal(t, tt.args.order.RemainingQuantity, tt.args.order.Quantity)
		})
	}
}

// Helpers para clonar pedidos e não compartilhar ponteiros entre casos
func validBuyClone(src *entity.Order) *entity.Order {
	cp := *src
	cp.Price = decimal.RequireFromString(src.Price.String())
	cp.Quantity = decimal.RequireFromString(src.Quantity.String())
	return &cp
}
func validSellClone(src *entity.Order) *entity.Order {
	cp := *src
	cp.Price = decimal.RequireFromString(src.Price.String())
	cp.Quantity = decimal.RequireFromString(src.Quantity.String())
	return &cp
}

func TestOrderUseCase_matchOrder(t *testing.T) {
	accountID := uuid.New()

	tests := []struct {
		name      string
		order     *entity.Order
		mockSetup func(or *repository.MockOrderRepository, o *entity.Order) []*entity.Order
		execSetup func(exec *MockTradeExecutor, o *entity.Order, matches []*entity.Order, captured *[]decimal.Decimal)
		wantErr   bool
	}{
		{
			name: "single partial fill",
			order: &entity.Order{
				AccountID:         accountID,
				InstrumentPair:    "BTC_BRL",
				OrderType:         string(entity.OrderTypeBuy),
				Price:             decimal.RequireFromString("100"),
				Quantity:          decimal.RequireFromString("1.0"),
				RemainingQuantity: decimal.RequireFromString("1.0"),
			},
			mockSetup: func(or *repository.MockOrderRepository, o *entity.Order) []*entity.Order {
				m1 := &entity.Order{
					AccountID:         uuid.New(),
					OrderType:         string(entity.OrderTypeSell),
					Price:             decimal.RequireFromString("99"),
					RemainingQuantity: decimal.RequireFromString("0.4"),
				}
				or.EXPECT().
					GetMatchingOrders(gomock.Any(), o.AccountID, o.InstrumentPair, "SELL", o.Price, true).
					Return([]*entity.Order{m1}, nil).
					Times(1)
				return []*entity.Order{m1}
			},
			execSetup: func(exec *MockTradeExecutor, o *entity.Order, matches []*entity.Order, captured *[]decimal.Decimal) {
				exec.EXPECT().
					Execute(gomock.Any(), o, matches[0], gomock.AssignableToTypeOf(decimal.Zero)).
					Return(nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "multiple fills until taker fully filled, then break (no 3rd call)",
			order: &entity.Order{
				AccountID:         accountID,
				InstrumentPair:    "BTC_BRL",
				OrderType:         string(entity.OrderTypeSell),
				Price:             decimal.RequireFromString("101"),
				Quantity:          decimal.RequireFromString("1.0"),
				RemainingQuantity: decimal.RequireFromString("1.0"),
			},
			mockSetup: func(or *repository.MockOrderRepository, o *entity.Order) []*entity.Order {
				m1 := &entity.Order{AccountID: uuid.New(), OrderType: string(entity.OrderTypeBuy), Price: decimal.RequireFromString("101"), RemainingQuantity: decimal.RequireFromString("0.4")}
				m2 := &entity.Order{AccountID: uuid.New(), OrderType: string(entity.OrderTypeBuy), Price: decimal.RequireFromString("102"), RemainingQuantity: decimal.RequireFromString("0.6")}
				m3 := &entity.Order{AccountID: uuid.New(), OrderType: string(entity.OrderTypeBuy), Price: decimal.RequireFromString("103"), RemainingQuantity: decimal.RequireFromString("0.5")}
				or.EXPECT().
					GetMatchingOrders(gomock.Any(), o.AccountID, o.InstrumentPair, "BUY", o.Price, false).
					Return([]*entity.Order{m1, m2, m3}, nil).
					Times(1)
				return []*entity.Order{m1, m2, m3}
			},
			execSetup: func(exec *MockTradeExecutor, o *entity.Order, matches []*entity.Order, captured *[]decimal.Decimal) {
				exec.EXPECT().
					Execute(gomock.Any(), o, gomock.Any(), gomock.AssignableToTypeOf(decimal.Zero)).
					Return(nil).
					Times(3)
			},
			wantErr: false,
		},
		{
			name: "repository error bubbles up",
			order: &entity.Order{
				AccountID:         accountID,
				InstrumentPair:    "BTC_BRL",
				OrderType:         string(entity.OrderTypeBuy),
				Price:             decimal.RequireFromString("100"),
				Quantity:          decimal.RequireFromString("1.0"),
				RemainingQuantity: decimal.RequireFromString("1.0"),
			},
			mockSetup: func(or *repository.MockOrderRepository, o *entity.Order) []*entity.Order {
				or.EXPECT().
					GetMatchingOrders(gomock.Any(), o.AccountID, o.InstrumentPair, "SELL", o.Price, true).
					Return(nil, errors.New("db error")).
					Times(1)
				return nil
			},
			execSetup: func(exec *MockTradeExecutor, o *entity.Order, matches []*entity.Order, captured *[]decimal.Decimal) {
			},
			wantErr: true,
		},
		{
			name: "no matches does nothing",
			order: &entity.Order{
				AccountID:         accountID,
				InstrumentPair:    "BTC_BRL",
				OrderType:         string(entity.OrderTypeBuy),
				Price:             decimal.RequireFromString("100"),
				Quantity:          decimal.RequireFromString("1.0"),
				RemainingQuantity: decimal.RequireFromString("1.0"),
			},
			mockSetup: func(or *repository.MockOrderRepository, o *entity.Order) []*entity.Order {
				or.EXPECT().
					GetMatchingOrders(gomock.Any(), o.AccountID, o.InstrumentPair, "SELL", o.Price, true).
					Return([]*entity.Order{}, nil).
					Times(1)
				return []*entity.Order{}
			},
			execSetup: func(exec *MockTradeExecutor, o *entity.Order, matches []*entity.Order, captured *[]decimal.Decimal) {
			},
			wantErr: false,
		},
		{
			name: "executor returns error, stops and returns error",
			order: &entity.Order{
				AccountID:         accountID,
				InstrumentPair:    "BTC_BRL",
				OrderType:         string(entity.OrderTypeBuy),
				Price:             decimal.RequireFromString("100"),
				Quantity:          decimal.RequireFromString("1.0"),
				RemainingQuantity: decimal.RequireFromString("1.0"),
			},
			mockSetup: func(or *repository.MockOrderRepository, o *entity.Order) []*entity.Order {
				m1 := &entity.Order{AccountID: uuid.New(), OrderType: string(entity.OrderTypeSell), Price: decimal.RequireFromString("100"), RemainingQuantity: decimal.RequireFromString("0.7")}
				or.EXPECT().
					GetMatchingOrders(gomock.Any(), o.AccountID, o.InstrumentPair, "SELL", o.Price, true).
					Return([]*entity.Order{m1}, nil).
					Times(1)
				return []*entity.Order{m1}
			},
			execSetup: func(exec *MockTradeExecutor, o *entity.Order, matches []*entity.Order, captured *[]decimal.Decimal) {
				exec.EXPECT().
					Execute(gomock.Any(), o, matches[0], gomock.AssignableToTypeOf(decimal.Zero)).
					Return(errors.New("exec failed")).
					Times(1)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			orderRepo := repository.NewMockOrderRepository(ctrl)
			matches := tt.mockSetup(orderRepo, tt.order)
			exec := NewMockTradeExecutor(ctrl)
			var captured []decimal.Decimal
			tt.execSetup(exec, tt.order, matches, &captured)

			db := newInMemoryDB(t)
			uc := &orderUseCase{
				log:             zap.NewNop().Sugar(),
				orderRepository: orderRepo,
				db:              db,
				executor:        exec,
			}

			tx := db.Begin()
			err := uc.matchOrder(tt.order, tx)

			if tt.wantErr {
				assert.Error(t, err)
				_ = tx.Rollback()
				return
			}

			assert.Nil(t, err)
			_ = tx.Rollback()
		})
	}
}
