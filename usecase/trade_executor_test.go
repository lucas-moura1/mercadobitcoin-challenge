package usecase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/lucas-moura1/mercadobitcoin-challenge/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestTradeExecutor_updateOrderStatus(t *testing.T) {
	tests := []struct {
		name        string
		qty         string
		remaining   string
		initialStat string
		wantStatus  string
		expectErr   bool
		repoErr     error
	}{
		{
			name:        "FILLED when remaining is zero",
			qty:         "1.0",
			remaining:   "0.0",
			initialStat: string(entity.OrderStatusPartial),
			wantStatus:  string(entity.OrderStatusFilled),
		},
		{
			name:        "OPEN when remaining equals quantity",
			qty:         "1.0",
			remaining:   "1.0",
			initialStat: string(entity.OrderStatusOpen),
			wantStatus:  string(entity.OrderStatusOpen),
		},
		{
			name:        "PARTIALLY_FILLED when 0 < remaining < quantity",
			qty:         "1.0",
			remaining:   "0.4",
			initialStat: string(entity.OrderStatusOpen),
			wantStatus:  string(entity.OrderStatusPartial),
		},
		{
			name:        "repository error returned and status not updated",
			qty:         "1.0",
			remaining:   "0.5",
			initialStat: string(entity.OrderStatusOpen),
			wantStatus:  string(entity.OrderStatusPartial),
			expectErr:   true,
			repoErr:     assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			orderRepo := repository.NewMockOrderRepository(ctrl)

			id := uuid.New()
			qty := decimal.RequireFromString(tt.qty)
			rem := decimal.RequireFromString(tt.remaining)

			o := &entity.Order{
				Base:              entity.Base{ID: id},
				Quantity:          qty,
				RemainingQuantity: rem,
				Status:            tt.initialStat,
			}

			orderRepo.EXPECT().
				UpdateRemainingAndStatus(gomock.Any(), id, rem, tt.wantStatus).
				Return(tt.repoErr).
				Times(1)

			exec := &tradeExecutor{
				log:       zap.NewNop().Sugar(),
				orderRepo: orderRepo,
			}

			err := exec.updateOrderStatus((*gorm.DB)(nil), o)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Equal(t, tt.initialStat, o.Status)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, o.Status)
		})
	}
}

func TestTradeExecutor_settle(t *testing.T) {
	type fields struct {
		orderType string
		buyerID   uuid.UUID
		sellerID  uuid.UUID
		price     decimal.Decimal
		qty       decimal.Decimal
	}

	tests := []struct {
		name      string
		f         fields
		mockSetup func(wr *repository.MockWalletRepository, f fields)
		wantErr   bool
	}{
		{
			name: "BUY flow - all legs succeed",
			f: fields{
				orderType: string(entity.OrderTypeBuy),
				buyerID:   uuid.New(),
				sellerID:  uuid.New(),
				price:     decimal.RequireFromString("200000"),
				qty:       decimal.RequireFromString("0.5"),
			},
			mockSetup: func(wr *repository.MockWalletRepository, f fields) {
				total := f.price.Mul(f.qty)
				gomock.InOrder(
					wr.EXPECT().SubtractFromBalance(nil, f.sellerID, "BTC", f.qty).Return(nil),
					wr.EXPECT().AddToBalance(nil, f.buyerID, "BTC", f.qty).Return(nil),
					wr.EXPECT().SubtractFromBalance(nil, f.buyerID, "BRL", total).Return(nil),
					wr.EXPECT().AddToBalance(nil, f.sellerID, "BRL", total).Return(nil),
				)
			},
		},
		{
			name: "SELL flow - roles swapped - all legs succeed",
			f: fields{
				orderType: string(entity.OrderTypeSell),
				buyerID:   uuid.New(),
				sellerID:  uuid.New(),
				price:     decimal.RequireFromString("199999"),
				qty:       decimal.RequireFromString("0.3"),
			},
			mockSetup: func(wr *repository.MockWalletRepository, f fields) {
				total := f.price.Mul(f.qty)
				gomock.InOrder(
					wr.EXPECT().SubtractFromBalance(nil, f.buyerID, "BTC", f.qty).Return(nil),
					wr.EXPECT().AddToBalance(nil, f.sellerID, "BTC", f.qty).Return(nil),
					wr.EXPECT().SubtractFromBalance(nil, f.sellerID, "BRL", total).Return(nil),
					wr.EXPECT().AddToBalance(nil, f.buyerID, "BRL", total).Return(nil),
				)
			},
		},
		{
			name: "error at base subtract stops flow",
			f: fields{
				orderType: string(entity.OrderTypeBuy),
				buyerID:   uuid.New(),
				sellerID:  uuid.New(),
				price:     decimal.RequireFromString("210000"),
				qty:       decimal.RequireFromString("0.1"),
			},
			mockSetup: func(wr *repository.MockWalletRepository, f fields) {
				wr.EXPECT().SubtractFromBalance(nil, f.sellerID, "BTC", f.qty).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "error at base add stops before quote legs",
			f: fields{
				orderType: string(entity.OrderTypeBuy),
				buyerID:   uuid.New(),
				sellerID:  uuid.New(),
				price:     decimal.RequireFromString("220000"),
				qty:       decimal.RequireFromString("0.2"),
			},
			mockSetup: func(wr *repository.MockWalletRepository, f fields) {
				wr.EXPECT().SubtractFromBalance(nil, f.sellerID, "BTC", f.qty).Return(nil)
				wr.EXPECT().AddToBalance(nil, f.buyerID, "BTC", f.qty).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "error at quote subtract stops before final add",
			f: fields{
				orderType: string(entity.OrderTypeBuy),
				buyerID:   uuid.New(),
				sellerID:  uuid.New(),
				price:     decimal.RequireFromString("230000"),
				qty:       decimal.RequireFromString("0.05"),
			},
			mockSetup: func(wr *repository.MockWalletRepository, f fields) {
				total := f.price.Mul(f.qty)
				wr.EXPECT().SubtractFromBalance(nil, f.sellerID, "BTC", f.qty).Return(nil)
				wr.EXPECT().AddToBalance(nil, f.buyerID, "BTC", f.qty).Return(nil)
				wr.EXPECT().SubtractFromBalance(nil, f.buyerID, "BRL", total).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "error at quote add returns error",
			f: fields{
				orderType: string(entity.OrderTypeBuy),
				buyerID:   uuid.New(),
				sellerID:  uuid.New(),
				price:     decimal.RequireFromString("240000"),
				qty:       decimal.RequireFromString("0.15"),
			},
			mockSetup: func(wr *repository.MockWalletRepository, f fields) {
				total := f.price.Mul(f.qty)
				wr.EXPECT().SubtractFromBalance(nil, f.sellerID, "BTC", f.qty).Return(nil)
				wr.EXPECT().AddToBalance(nil, f.buyerID, "BTC", f.qty).Return(nil)
				wr.EXPECT().SubtractFromBalance(nil, f.buyerID, "BRL", total).Return(nil)
				wr.EXPECT().AddToBalance(nil, f.sellerID, "BRL", total).Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			walletRepo := repository.NewMockWalletRepository(ctrl)

			tt.mockSetup(walletRepo, tt.f)

			exec := &tradeExecutor{
				log:        zap.NewNop().Sugar(),
				walletRepo: walletRepo,
			}

			order := &entity.Order{
				AccountID:      tt.f.buyerID,
				InstrumentPair: "BTC_BRL",
				OrderType:      tt.f.orderType,
				Price:          tt.f.price,
			}
			matching := &entity.Order{
				AccountID:      tt.f.sellerID,
				InstrumentPair: "BTC_BRL",
				OrderType:      tt.f.orderType,
				Price:          tt.f.price,
			}

			err := exec.settle(nil, order, matching, tt.f.qty)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}

func TestTradeExecutor_Execute_TableDriven(t *testing.T) {
	type args struct {
		matchingType   string
		price          string
		qty            string
		matchingRemain string
		orderRemain    string
	}

	tests := []struct {
		name    string
		args    args
		setup   func(or *repository.MockOrderRepository, wr *repository.MockWalletRepository, tr *repository.MockTradeRepository, order, matching *entity.Order, qty, price decimal.Decimal)
		wantErr bool
	}{
		{
			name: "success - BUY matching (full fill)",
			args: args{
				matchingType:   string(entity.OrderTypeBuy),
				price:          "200000",
				qty:            "0.5",
				matchingRemain: "0.5",
				orderRemain:    "0.5",
			},
			setup: func(or *repository.MockOrderRepository, wr *repository.MockWalletRepository, tr *repository.MockTradeRepository, order, matching *entity.Order, qty, price decimal.Decimal) {
				tr.EXPECT().
					Create(gomock.Nil(), gomock.Any()).
					Return(nil).Times(1)

				or.EXPECT().UpdateRemainingAndStatus(gomock.Nil(), matching.ID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				or.EXPECT().UpdateRemainingAndStatus(gomock.Nil(), order.ID, gomock.Any(), gomock.Any()).Return(nil).Times(1)

				wr.EXPECT().SubtractFromBalance(gomock.Nil(), order.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().AddToBalance(gomock.Nil(), matching.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().SubtractFromBalance(gomock.Nil(), matching.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().AddToBalance(gomock.Nil(), order.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "success - SELL matching (full fill)",
			args: args{
				matchingType:   string(entity.OrderTypeSell),
				price:          "201000",
				qty:            "0.3",
				matchingRemain: "0.3",
				orderRemain:    "0.3",
			},
			setup: func(or *repository.MockOrderRepository, wr *repository.MockWalletRepository, tr *repository.MockTradeRepository, order, matching *entity.Order, qty, price decimal.Decimal) {
				tr.EXPECT().
					Create(gomock.Nil(), gomock.Any()).
					Return(nil).Times(1)

				or.EXPECT().UpdateRemainingAndStatus(gomock.Nil(), matching.ID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				or.EXPECT().UpdateRemainingAndStatus(gomock.Nil(), order.ID, gomock.Any(), gomock.Any()).Return(nil).Times(1)

				wr.EXPECT().SubtractFromBalance(gomock.Nil(), matching.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().AddToBalance(gomock.Nil(), order.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().SubtractFromBalance(gomock.Nil(), order.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().AddToBalance(gomock.Nil(), matching.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "partial fill - matching remains, order filled",
			args: args{
				matchingType:   string(entity.OrderTypeBuy),
				price:          "199999",
				qty:            "0.4",
				matchingRemain: "1.0",
				orderRemain:    "0.4",
			},
			setup: func(or *repository.MockOrderRepository, wr *repository.MockWalletRepository, tr *repository.MockTradeRepository, order, matching *entity.Order, qty, price decimal.Decimal) {
				tr.EXPECT().Create(gomock.Nil(), gomock.Any()).Return(nil).Times(1)
				or.EXPECT().UpdateRemainingAndStatus(gomock.Nil(), matching.ID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				or.EXPECT().UpdateRemainingAndStatus(gomock.Nil(), order.ID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().SubtractFromBalance(gomock.Nil(), order.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().AddToBalance(gomock.Nil(), matching.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().SubtractFromBalance(gomock.Nil(), matching.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				wr.EXPECT().AddToBalance(gomock.Nil(), order.AccountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "error - trade creation fails",
			args: args{
				matchingType:   string(entity.OrderTypeBuy),
				price:          "100",
				qty:            "0.1",
				matchingRemain: "0.1",
				orderRemain:    "0.1",
			},
			setup: func(or *repository.MockOrderRepository, wr *repository.MockWalletRepository, tr *repository.MockTradeRepository, order, matching *entity.Order, qty, price decimal.Decimal) {
				tr.EXPECT().Create(gomock.Nil(), gomock.Any()).Return(assert.AnError).Times(1)
			},
			wantErr: true,
		},
		{
			name: "error - first UpdateRemainingAndStatus fails (no second update, no settle)",
			args: args{
				matchingType:   string(entity.OrderTypeBuy),
				price:          "100",
				qty:            "0.2",
				matchingRemain: "0.2",
				orderRemain:    "0.2",
			},
			setup: func(or *repository.MockOrderRepository, wr *repository.MockWalletRepository, tr *repository.MockTradeRepository, order, matching *entity.Order, qty, price decimal.Decimal) {
				tr.EXPECT().Create(gomock.Nil(), gomock.Any()).Return(nil).Times(1)
				or.EXPECT().UpdateRemainingAndStatus(gomock.Nil(), order.ID, gomock.Any(), gomock.Any()).Return(assert.AnError).Times(1)
			},
			wantErr: true,
		},
		{
			name: "error - second UpdateRemainingAndStatus fails (no settle)",
			args: args{
				matchingType:   string(entity.OrderTypeBuy),
				price:          "100",
				qty:            "0.25",
				matchingRemain: "0.25",
				orderRemain:    "0.25",
			},
			setup: func(or *repository.MockOrderRepository, wr *repository.MockWalletRepository, tr *repository.MockTradeRepository, order, matching *entity.Order, qty, price decimal.Decimal) {
				tr.EXPECT().Create(gomock.Nil(), gomock.Any()).Return(nil).Times(1)
				or.EXPECT().UpdateRemainingAndStatus(gomock.Nil(), order.ID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
				or.EXPECT().UpdateRemainingAndStatus(gomock.Nil(), matching.ID, gomock.Any(), gomock.Any()).Return(assert.AnError).Times(1)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			orderRepo := repository.NewMockOrderRepository(ctrl)
			walletRepo := repository.NewMockWalletRepository(ctrl)
			tradeRepo := repository.NewMockTradeRepository(ctrl)

			price := decimal.RequireFromString(tt.args.price)
			qty := decimal.RequireFromString(tt.args.qty)

			matching := &entity.Order{
				Base:              entity.Base{ID: uuid.New()},
				AccountID:         uuid.New(),
				InstrumentPair:    "BTC_BRL",
				OrderType:         tt.args.matchingType,
				Price:             price,
				Quantity:          decimal.RequireFromString(tt.args.matchingRemain),
				RemainingQuantity: decimal.RequireFromString(tt.args.matchingRemain),
				Status:            string(entity.OrderStatusOpen),
			}

			orderType := string(entity.OrderTypeSell)
			if tt.args.matchingType == string(entity.OrderTypeSell) {
				orderType = string(entity.OrderTypeBuy)
			}
			order := &entity.Order{
				Base:              entity.Base{ID: uuid.New()},
				AccountID:         uuid.New(),
				InstrumentPair:    "BTC_BRL",
				OrderType:         orderType,
				Price:             price,
				Quantity:          decimal.RequireFromString(tt.args.orderRemain),
				RemainingQuantity: decimal.RequireFromString(tt.args.orderRemain),
				Status:            string(entity.OrderStatusOpen),
			}

			tt.setup(orderRepo, walletRepo, tradeRepo, order, matching, qty, price)

			exec := &tradeExecutor{
				log:        zap.NewNop().Sugar(),
				orderRepo:  orderRepo,
				walletRepo: walletRepo,
				tradeRepo:  tradeRepo,
			}

			err := exec.Execute(nil, order, matching, qty)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}
