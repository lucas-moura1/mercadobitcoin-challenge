package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/lucas-moura1/mercadobitcoin-challenge/usecase"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestAccountHandler_GetAccountBalance(t *testing.T) {
	tests := []struct {
		name       string
		pathValue  string
		setupMock  func(m *usecase.MockAccountUseCase, id string)
		wantStatus int
	}{
		{
			name:      "success returns 200 and JSON",
			pathValue: uuid.New().String(),
			setupMock: func(m *usecase.MockAccountUseCase, id string) {
				uid, _ := uuid.Parse(id)
				m.EXPECT().GetAccountBalance(uid).Return([]*entity.Wallet{
					{
						AccountID:   uid,
						AssetSymbol: "BTC",
						Balance:     decimal.RequireFromString("0.5"),
					},
				}, nil).Times(1)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			pathValue:  "test",
			setupMock:  func(m *usecase.MockAccountUseCase, id string) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "usecase error returns 500",
			pathValue: uuid.New().String(),
			setupMock: func(m *usecase.MockAccountUseCase, id string) {
				uid, _ := uuid.Parse(id)
				m.EXPECT().GetAccountBalance(uid).Return(nil, assert.AnError).Times(1)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:      "not found returns 404",
			pathValue: uuid.New().String(),
			setupMock: func(m *usecase.MockAccountUseCase, id string) {
				uid, _ := uuid.Parse(id)
				m.EXPECT().GetAccountBalance(uid).Return(nil, nil).Times(1)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := usecase.NewMockAccountUseCase(ctrl)

			h := NewAccountHandler(zap.NewNop().Sugar(), mockUC)

			req := httptest.NewRequest(http.MethodGet, "/accounts/{id}/balance", nil)
			req.SetPathValue("id", tt.pathValue)
			respWriter := httptest.NewRecorder()

			tt.setupMock(mockUC, tt.pathValue)

			h.GetAccountBalance(respWriter, req)
			assert.Equal(t, tt.wantStatus, respWriter.Code)
		})
	}
}
