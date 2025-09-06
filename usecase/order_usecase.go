package usecase

import (
	"errors"
	"sort"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/lucas-moura1/mercadobitcoin-challenge/repository"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type orderUseCase struct {
	log              *zap.SugaredLogger
	orderRepository  repository.OrderRepository
	walletRepository repository.WalletRepository
	tradeRepository  repository.TradeRepository
	db               *gorm.DB
	executor         TradeExecutor
}

func NewOrderUseCase(
	log *zap.SugaredLogger,
	orderRepo repository.OrderRepository,
	walletRepo repository.WalletRepository,
	tradeRepo repository.TradeRepository,
	db *gorm.DB,
) OrderUseCase {
	return &orderUseCase{
		log:              log,
		orderRepository:  orderRepo,
		walletRepository: walletRepo,
		tradeRepository:  tradeRepo,
		db:               db,
		executor:         NewTradeExecutor(log, orderRepo, walletRepo, tradeRepo),
	}
}

func (u *orderUseCase) CreateOrder(order *entity.Order) error {
	u.log.Infow("creating new order",
		"account_id", order.AccountID,
		"type", order.OrderType,
		"instrument_pair", order.InstrumentPair,
	)

	tx := u.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := order.Validate(); err != nil {
		u.log.Errorw("invalid order", "error", err)
		return err
	}

	if err := u.checkWalletBalance(order, tx); err != nil {
		tx.Rollback()
		return err
	}

	order.Status = string(entity.OrderStatusOpen)
	order.RemainingQuantity = order.Quantity

	if err := u.orderRepository.Create(tx, order); err != nil {
		tx.Rollback()
		return err
	}

	if err := u.matchOrder(order, tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (u *orderUseCase) matchOrder(order *entity.Order, tx *gorm.DB) error {
	u.log.Infow("matching order",
		"order_id", order.ID,
		"type", order.OrderType,
		"price", order.Price,
	)

	oppositeOrderType := "SELL"
	if order.OrderType == "SELL" {
		oppositeOrderType = "BUY"
	}
	matchingOrders, err := u.orderRepository.GetMatchingOrders(
		tx,
		order.AccountID,
		order.InstrumentPair,
		oppositeOrderType,
		order.Price,
		order.OrderType == "BUY",
	)
	if err != nil {
		return err
	}

	if len(matchingOrders) == 0 {
		return nil
	}

	for _, matchingOrder := range matchingOrders {
		qty := decimal.Min(order.RemainingQuantity, matchingOrder.RemainingQuantity)
		if err := u.executor.Execute(tx, order, matchingOrder, qty); err != nil {
			return err
		}
		if order.RemainingQuantity.IsZero() {
			break
		}
	}
	return nil
}

func (u *orderUseCase) CancelOrder(id uuid.UUID) error {
	u.log.Infow("canceling order", "id", id)

	order, err := u.orderRepository.GetByID(id, string(entity.OrderStatusOpen))
	if err != nil {
		return err
	}
	if order == nil {
		return nil
	}

	if err := u.orderRepository.UpdateStatus(id, string(entity.OrderStatusCancelled)); err != nil {
		return err
	}

	return nil
}

func (u *orderUseCase) checkWalletBalance(order *entity.Order, tx *gorm.DB) error {
	requiredAsset, requiredAmount := order.GetRequiredAssetAndAmount()

	wallet, err := u.walletRepository.GetByAccountAndAsset(tx, order.AccountID, requiredAsset)
	if err != nil {
		return err
	}

	if wallet == nil {
		return errors.New("wallet not found for required asset")
	}

	if wallet.Balance.LessThan(requiredAmount) {
		u.log.Errorw("insufficient balance",
			"account_id", order.AccountID,
			"asset", requiredAsset)
		return errors.New("insufficient balance")
	}

	return nil
}

func (u *orderUseCase) GetOrderBook(instrumentPair string) (*OrderBook, error) {
	u.log.Infow("getting order book", "instrument_pair", instrumentPair)

	if !entity.IsValidInstrumentPair(instrumentPair) {
		return nil, entity.ErrInvalidPairFormat
	}

	orders, err := u.orderRepository.GetOpenOrdersByInstrumentPair(instrumentPair)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, nil
	}

	orderBook := &OrderBook{
		InstrumentPair: instrumentPair,
		Bids:           make([]*OrderBookEntry, 0),
		Asks:           make([]*OrderBookEntry, 0),
	}

	bidsMap := make(map[string]decimal.Decimal)
	asksMap := make(map[string]decimal.Decimal)

	for _, order := range orders {
		if order.OrderType == "BUY" {
			bidsMap[order.Price.String()] = bidsMap[order.Price.String()].Add(order.RemainingQuantity)
		} else {
			asksMap[order.Price.String()] = asksMap[order.Price.String()].Add(order.RemainingQuantity)
		}
	}

	bidPrices := make([]decimal.Decimal, 0, len(bidsMap))
	for p := range bidsMap {
		bidPrices = append(bidPrices, decimal.RequireFromString(p))
	}
	sort.Slice(bidPrices, func(i, j int) bool {
		return bidPrices[i].GreaterThan(bidPrices[j])
	})
	for _, p := range bidPrices {
		orderBook.Bids = append(orderBook.Bids, &OrderBookEntry{
			Price:    p,
			Quantity: bidsMap[p.String()],
		})
	}

	askPrices := make([]decimal.Decimal, 0, len(asksMap))
	for p := range asksMap {
		askPrices = append(askPrices, decimal.RequireFromString(p))
	}
	sort.Slice(askPrices, func(i, j int) bool {
		return askPrices[i].LessThan(askPrices[j])
	})
	for _, p := range askPrices {
		orderBook.Asks = append(orderBook.Asks, &OrderBookEntry{
			Price:    p,
			Quantity: asksMap[p.String()],
		})
	}

	return orderBook, nil
}
