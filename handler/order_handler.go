package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/lucas-moura1/mercadobitcoin-challenge/usecase"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type orderHandler struct {
	log          *zap.SugaredLogger
	orderUseCase usecase.OrderUseCase
}

func NewOrderHandler(log *zap.SugaredLogger, orderUseCase usecase.OrderUseCase) *orderHandler {
	return &orderHandler{log: log, orderUseCase: orderUseCase}
}

type CreateOrderRequest struct {
	AccountID      uuid.UUID `json:"account_id"`
	InstrumentPair string    `json:"instrument_pair"`
	OrderType      string    `json:"order_type"`
	Price          string    `json:"price"`
	Quantity       string    `json:"quantity"`
}

type CreateOrderResponse struct {
	OrderID        uuid.UUID `json:"order_id"`
	InstrumentPair string    `json:"instrument_pair"`
	OrderType      string    `json:"order_type"`
	Price          string    `json:"price"`
	Quantity       string    `json:"quantity"`
	Status         string    `json:"status"`
}

func (h *orderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	req := new(CreateOrderRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		h.log.Errorw("failed to decode request", "error", err)
		errorHandler(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	price, err := decimal.NewFromString(req.Price)
	if err != nil {
		h.log.Errorw("invalid price format", "error", err)
		errorHandler(w, http.StatusBadRequest, "Invalid price format")
		return
	}

	quantity, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		h.log.Errorw("invalid quantity format", "error", err)
		errorHandler(w, http.StatusBadRequest, "Invalid quantity format")
		return
	}

	order := &entity.Order{
		AccountID:      req.AccountID,
		InstrumentPair: req.InstrumentPair,
		OrderType:      req.OrderType,
		Price:          price,
		Quantity:       quantity,
	}

	if err := h.orderUseCase.CreateOrder(order); err != nil {
		h.log.Errorw("failed to create order", "error", err)
		errorHandler(w, http.StatusBadRequest, err.Error())
		return
	}

	response := &CreateOrderResponse{
		OrderID:        order.ID,
		InstrumentPair: order.InstrumentPair,
		OrderType:      order.OrderType,
		Price:          order.Price.String(),
		Quantity:       order.Quantity.String(),
		Status:         order.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *orderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	orderID, err := uuid.Parse(id)
	if err != nil {
		h.log.Errorw("invalid order id", "error", err)
		errorHandler(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	if err := h.orderUseCase.CancelOrder(orderID); err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

type OrderBookResponse struct {
	InstrumentPair string           `json:"instrument_pair"`
	Bids           []OrderBookLevel `json:"bids"`
	Asks           []OrderBookLevel `json:"asks"`
}

type OrderBookLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}
func (h *orderHandler) GetOrderBook(w http.ResponseWriter, r *http.Request) {
	instrumentPair := r.PathValue("instrument_pair")
	orderBook, err := h.orderUseCase.GetOrderBook(instrumentPair)
	if err != nil {
		h.log.Errorw("failed to get order book",
			"instrument_pair", instrumentPair,
			"error", err,
		)
		if errors.Is(err, entity.ErrInvalidPairFormat) {
			errorHandler(w, http.StatusBadRequest, err.Error())
			return
		}
		errorHandler(w, http.StatusInternalServerError, err.Error())
		return
	}

	if orderBook == nil {
		errorHandler(w, http.StatusNotFound, "Order book not found")
		return
	}

	response := OrderBookResponse{
		InstrumentPair: orderBook.InstrumentPair,
		Bids:           make([]OrderBookLevel, len(orderBook.Bids)),
		Asks:           make([]OrderBookLevel, len(orderBook.Asks)),
	}

	for i, bid := range orderBook.Bids {
		response.Bids[i] = OrderBookLevel{
			Price:    bid.Price.String(),
			Quantity: bid.Quantity.String(),
		}
	}

	for i, ask := range orderBook.Asks {
		response.Asks[i] = OrderBookLevel{
			Price:    ask.Price.String(),
			Quantity: ask.Quantity.String(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
