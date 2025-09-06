package repository

import (
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type tradeRepository struct {
	log *zap.SugaredLogger
}

func NewTradeRepository(log *zap.SugaredLogger) TradeRepository {
	return &tradeRepository{log: log}
}

func (r *tradeRepository) Create(tx *gorm.DB, trade *entity.Trade) error {
	r.log.Debugw("creating trade",
		"buyer_order_id", trade.BuyerOrderID,
		"seller_order_id", trade.SellerOrderID,
		"price", trade.Price,
		"quantity", trade.Quantity,
	)

	if err := tx.Create(trade).Error; err != nil {
		r.log.Errorw("failed to create trade", "error", err)
		return err
	}

	return nil
}
