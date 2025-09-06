package usecase

import (
	"strings"

	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/lucas-moura1/mercadobitcoin-challenge/repository"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type tradeExecutor struct {
	log        *zap.SugaredLogger
	orderRepo  repository.OrderRepository
	walletRepo repository.WalletRepository
	tradeRepo  repository.TradeRepository
}

func NewTradeExecutor(
	log *zap.SugaredLogger,
	orderRepo repository.OrderRepository,
	walletRepo repository.WalletRepository,
	tradeRepo repository.TradeRepository,
) TradeExecutor {
	return &tradeExecutor{log: log, orderRepo: orderRepo, walletRepo: walletRepo, tradeRepo: tradeRepo}
}

func (e *tradeExecutor) Execute(tx *gorm.DB, order, matchingOrder *entity.Order, qty decimal.Decimal) error {
	buyID, sellID := order.ID, matchingOrder.ID
	if order.OrderType == "SELL" {
		buyID, sellID = matchingOrder.ID, order.ID
	}
	trade := &entity.Trade{
		BuyerOrderID:  buyID,
		SellerOrderID: sellID,
		Price:         matchingOrder.Price,
		Quantity:      qty,
	}
	if err := e.tradeRepo.Create(tx, trade); err != nil {
		return err
	}

	e.log.Debugw("executed trade", "trade_id", trade.ID, "quantity", qty, "price", matchingOrder.Price)

	order.RemainingQuantity = order.RemainingQuantity.Sub(qty)
	matchingOrder.RemainingQuantity = matchingOrder.RemainingQuantity.Sub(qty)

	if err := e.updateOrderStatus(tx, order); err != nil {
		return err
	}
	if err := e.updateOrderStatus(tx, matchingOrder); err != nil {
		return err
	}

	e.log.Debugw("updated orders after trade")

	return e.settle(tx, order, matchingOrder, qty)
}

func (e *tradeExecutor) updateOrderStatus(tx *gorm.DB, o *entity.Order) error {
	var newStatus string
	switch {
	case o.RemainingQuantity.IsZero():
		newStatus = string(entity.OrderStatusFilled)
	case o.RemainingQuantity.Equal(o.Quantity):
		newStatus = string(entity.OrderStatusOpen)
	default:
		newStatus = string(entity.OrderStatusPartial)
	}

	if err := e.orderRepo.UpdateRemainingAndStatus(tx, o.ID, o.RemainingQuantity, newStatus); err != nil {
		return err
	}

	o.Status = newStatus
	return nil
}

func (e *tradeExecutor) settle(tx *gorm.DB, order, matchingOrder *entity.Order, qty decimal.Decimal) error {
	parts := strings.Split(order.InstrumentPair, "_")
	base, quote := parts[0], parts[1]

	buyer, seller := order, matchingOrder
	if order.OrderType == "SELL" {
		buyer, seller = matchingOrder, order
	}

	total := matchingOrder.Price.Mul(qty)

	if err := e.walletRepo.SubtractFromBalance(tx, seller.AccountID, base, qty); err != nil {
		return err
	}
	if err := e.walletRepo.AddToBalance(tx, buyer.AccountID, base, qty); err != nil {
		return err
	}

	if err := e.walletRepo.SubtractFromBalance(tx, buyer.AccountID, quote, total); err != nil {
		return err
	}
	if err := e.walletRepo.AddToBalance(tx, seller.AccountID, quote, total); err != nil {
		return err
	}

	e.log.Debugw("settled trade")
	return nil
}
