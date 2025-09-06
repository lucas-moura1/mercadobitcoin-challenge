package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/lucas-moura1/mercadobitcoin-challenge/usecase"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestOrderHandler_CancelOrder(t *testing.T) {
	tests := []struct {
		name       string
		pathValue  string
		setupMock  func(m *usecase.MockOrderUseCase, id string)
		wantStatus int
	}{
		{
			name:      "success returns 200",
			pathValue: uuid.New().String(),
			setupMock: func(m *usecase.MockOrderUseCase, id string) {
				uid, _ := uuid.Parse(id)
				m.EXPECT().CancelOrder(uid).Return(nil).Times(1)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			pathValue:  "not-a-uuid",
			setupMock:  func(m *usecase.MockOrderUseCase, id string) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "usecase error returns 500",
			pathValue: uuid.New().String(),
			setupMock: func(m *usecase.MockOrderUseCase, id string) {
				uid, _ := uuid.Parse(id)
				m.EXPECT().CancelOrder(uid).Return(assert.AnError).Times(1)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := usecase.NewMockOrderUseCase(ctrl)
			h := NewOrderHandler(zap.NewNop().Sugar(), mockUC)

			req := httptest.NewRequest(http.MethodPost, "/orders/{id}/cancel", nil)
			req.SetPathValue("id", tt.pathValue)
			respWriter := httptest.NewRecorder()

			tt.setupMock(mockUC, tt.pathValue)

			h.CancelOrder(respWriter, req)

			assert.Equal(t, tt.wantStatus, respWriter.Code)
		})
	}
}

func TestOrderHandler_GetOrderBook(t *testing.T) {
	tests := []struct {
		name       string
		pair       string
		mockSetup  func(m *usecase.MockOrderUseCase, pair string)
		wantStatus int
	}{
		{
			name: "invalid instrument pair returns 400",
			pair: "BTCBRL",
			mockSetup: func(m *usecase.MockOrderUseCase, pair string) {
				m.EXPECT().GetOrderBook(pair).Return(nil, entity.ErrInvalidPairFormat).Times(1)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "usecase error returns 500",
			pair: "BTC_BRL",
			mockSetup: func(m *usecase.MockOrderUseCase, pair string) {
				m.EXPECT().GetOrderBook(pair).Return(nil, assert.AnError).Times(1)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "not found (nil orderbook) returns 404",
			pair: "BTC_BRL",
			mockSetup: func(m *usecase.MockOrderUseCase, pair string) {
				m.EXPECT().GetOrderBook(pair).Return(nil, nil).Times(1)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "success returns 200 and body",
			pair: "BTC_BRL",
			mockSetup: func(m *usecase.MockOrderUseCase, pair string) {
				ob := &usecase.OrderBook{
					InstrumentPair: pair,
					Bids: []*usecase.OrderBookEntry{
						{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1.4")},
						{Price: decimal.RequireFromString("99"), Quantity: decimal.RequireFromString("2.0")},
					},
					Asks: []*usecase.OrderBookEntry{
						{Price: decimal.RequireFromString("101"), Quantity: decimal.RequireFromString("0.8")},
						{Price: decimal.RequireFromString("103"), Quantity: decimal.RequireFromString("0.2")},
					},
				}
				m.EXPECT().GetOrderBook(pair).Return(ob, nil).Times(1)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := usecase.NewMockOrderUseCase(ctrl)
			h := NewOrderHandler(zap.NewNop().Sugar(), mockUC)

			if tt.mockSetup != nil {
				tt.mockSetup(mockUC, tt.pair)
			}

			req := httptest.NewRequest(http.MethodGet, "/orders/{instrument_pair}", nil)
			req.SetPathValue("instrument_pair", tt.pair)
			respWriter := httptest.NewRecorder()

			h.GetOrderBook(respWriter, req)

			assert.Equal(t, tt.wantStatus, respWriter.Code)
			if respWriter.Code == http.StatusOK {
				assert.Equal(t, "application/json", respWriter.Header().Get("Content-Type"))
				var resp OrderBookResponse
				err := json.Unmarshal(respWriter.Body.Bytes(), &resp)
				assert.NoError(t, err)

				assert.Equal(t, "BTC_BRL", resp.InstrumentPair)
				if assert.Len(t, resp.Bids, 2) {
					assert.Equal(t, "100", resp.Bids[0].Price)
					assert.Equal(t, "1.4", resp.Bids[0].Quantity)
					assert.Equal(t, "99", resp.Bids[1].Price)
					assert.Equal(t, "2", resp.Bids[1].Quantity)
				}
				if assert.Len(t, resp.Asks, 2) {
					assert.Equal(t, "101", resp.Asks[0].Price)
					assert.Equal(t, "0.8", resp.Asks[0].Quantity)
					assert.Equal(t, "103", resp.Asks[1].Price)
					assert.Equal(t, "0.2", resp.Asks[1].Quantity)
				}
			}
		})
	}
}

func TestOrderHandler_CreateOrder(t *testing.T) {
	uid := uuid.New().String()

	tests := []struct {
		name       string
		body       string
		mockSetup  func(m *usecase.MockOrderUseCase)
		wantStatus int
	}{
		{
			name: "success returns 201 and response body",
			body: `{"account_id":"` + uid + `","instrument_pair":"BTC_BRL","order_type":"buy","price":"200000.00","quantity":"0.50"}`,
			mockSetup: func(m *usecase.MockOrderUseCase) {
				m.EXPECT().
					CreateOrder(gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid JSON body returns 400",
			body:       "{",
			mockSetup:  func(m *usecase.MockOrderUseCase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid price format returns 400",
			body:       `{"account_id":"` + uid + `","instrument_pair":"BTC_BRL","order_type":"buy","price":"abc","quantity":"0.5"}`,
			mockSetup:  func(m *usecase.MockOrderUseCase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid quantity format returns 400",
			body:       `{"account_id":"` + uid + `","instrument_pair":"BTC_BRL","order_type":"sell","price":"200000","quantity":"x.y"}`,
			mockSetup:  func(m *usecase.MockOrderUseCase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "usecase returns error returns 400",
			body: `{"account_id":"` + uid + `","instrument_pair":"BTC_BRL","order_type":"buy","price":"200000","quantity":"0.5"}`,
			mockSetup: func(m *usecase.MockOrderUseCase) {
				m.EXPECT().
					CreateOrder(gomock.Any()).
					Return(assert.AnError).
					Times(1)
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := usecase.NewMockOrderUseCase(ctrl)
			h := NewOrderHandler(zap.NewNop().Sugar(), mockUC)

			tt.mockSetup(mockUC)

			req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			respWriter := httptest.NewRecorder()

			h.CreateOrder(respWriter, req)

			assert.Equal(t, tt.wantStatus, respWriter.Code)
			if respWriter.Code == http.StatusCreated {
				assert.Equal(t, "application/json", respWriter.Header().Get("Content-Type"))

				var resp CreateOrderResponse
				err := json.Unmarshal(respWriter.Body.Bytes(), &resp)
				assert.NoError(t, err)

				assert.IsType(t, uuid.UUID{}, resp.OrderID)
				assert.Equal(t, "BTC_BRL", resp.InstrumentPair)
				assert.Equal(t, "buy", resp.OrderType)
				assert.Equal(t, "200000", resp.Price)
				assert.Equal(t, "0.5", resp.Quantity)
			}
		})
	}
}
