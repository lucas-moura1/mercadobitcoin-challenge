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
)

func TestAccountUseCase_GetAccountBalance(t *testing.T) {
	accountID := uuid.New()

	tests := []struct {
		name        string
		setupMock   func(m *repository.MockWalletRepository)
		wantLen     int
		wantNilResp bool
		wantErr     bool
	}{
		{
			name: "success with wallets",
			setupMock: func(m *repository.MockWalletRepository) {
				m.EXPECT().GetByAccountID(accountID).Return([]*entity.Wallet{
					{AccountID: accountID, AssetSymbol: "BTC", Balance: decimal.RequireFromString("0.5")},
					{AccountID: accountID, AssetSymbol: "BRL", Balance: decimal.RequireFromString("1000")},
				}, nil)
			},
			wantLen:     2,
			wantNilResp: false,
			wantErr:     false,
		},
		{
			name: "empty slice returns nil and no error",
			setupMock: func(m *repository.MockWalletRepository) {
				m.EXPECT().GetByAccountID(accountID).Return(nil, nil)
			},
			wantLen:     0,
			wantNilResp: true,
			wantErr:     false,
		},
		{
			name: "repository error",
			setupMock: func(m *repository.MockWalletRepository) {
				m.EXPECT().GetByAccountID(accountID).Return(nil, errors.New("database error"))
			},
			wantLen:     0,
			wantNilResp: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockWalletRepo := repository.NewMockWalletRepository(ctrl)

			tt.setupMock(mockWalletRepo)
			uc := NewAccountUseCase(zap.NewNop().Sugar(), mockWalletRepo)
			got, err := uc.GetAccountBalance(accountID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			if tt.wantNilResp {
				assert.Nil(t, got)
				return
			}
			assert.NotNil(t, got)
			assert.Len(t, got, tt.wantLen)
			assert.Equal(t, "BTC", got[0].AssetSymbol)
		})
	}
}
