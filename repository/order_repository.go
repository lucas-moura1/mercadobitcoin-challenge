package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type orderRepository struct {
	log *zap.SugaredLogger
	db  *gorm.DB
}

func NewOrderRepository(log *zap.SugaredLogger, db *gorm.DB) OrderRepository {
	return &orderRepository{log: log, db: db}
}

func (r *orderRepository) Create(tx *gorm.DB, order *entity.Order) error {
	r.log.Debugw("creating order",
		"account_id", order.AccountID,
		"type", order.OrderType,
		"instrument_pair", order.InstrumentPair,
	)

	db := r.db
	if tx != nil {
		db = tx
	}

	if err := db.Create(order).Error; err != nil {
		r.log.Errorw("failed to create order", "error", err)
		return err
	}

	return nil
}

func (r *orderRepository) GetOpenOrdersByInstrumentPair(instrumentPair string) ([]*entity.Order, error) {
	var orders []*entity.Order

	err := r.db.Where("instrument_pair = ? AND status = ?",
		instrumentPair, string(entity.OrderStatusOpen)).Find(&orders).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warnw("no open orders found", "instrument_pair", instrumentPair)
			return nil, nil
		}
		r.log.Errorw("failed to get open orders",
			"instrument_pair", instrumentPair,
			"error", err,
		)
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) GetByID(id uuid.UUID, status ...string) (*entity.Order, error) {

	whereCondition := "id = ?"
	if len(status) > 0 {
		whereCondition += " AND status IN ?"
	}
	order := new(entity.Order)
	err := r.db.Where(whereCondition, id, status).First(order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warnw("order not found", "id", id)
			return nil, nil
		}
		r.log.Errorw("failed to get order", "id", id, "error", err)
		return nil, err
	}

	return order, nil
}

func (r *orderRepository) UpdateStatus(id uuid.UUID, status string) error {
	r.log.Debugw("updating order status",
		"id", id,
		"status", status,
	)

	if err := r.db.Model(&entity.Order{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		r.log.Errorw("failed to update order status",
			"id", id,
			"error", err,
		)
		return err
	}

	return nil
}

func (r *orderRepository) UpdateRemainingAndStatus(tx *gorm.DB, id uuid.UUID, quantity decimal.Decimal, status string) error {
	r.log.Debugw("updating order remaining quantity and status",
		"id", id,
		"quantity", quantity,
		"status", status,
	)

	db := r.db
	if tx != nil {
		db = tx
	}

	if err := db.Model(&entity.Order{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"remaining_quantity": quantity,
			"status":             status,
		}).Error; err != nil {
		r.log.Errorw("failed to update order remaining quantity and status",
			"id", id,
			"error", err,
		)
		return err
	}

	return nil
}

func (r *orderRepository) GetMatchingOrders(
	tx *gorm.DB,
	accountID uuid.UUID,
	instrumentPair string,
	orderType string,
	price decimal.Decimal,
	isBuyOrder bool,
) ([]*entity.Order, error) {
	var orders []*entity.Order

	db := r.db
	if tx != nil {
		db = tx
	}

	query := db.Where("instrument_pair = ? AND order_type = ? AND status IN (?) AND account_id <> ?",
		instrumentPair, orderType, []string{string(entity.OrderStatusOpen), string(entity.OrderStatusPartial)}, accountID)

	if isBuyOrder {
		query = query.Where("price <= ?", price).Order("price ASC, created_at ASC")
	} else {
		query = query.Where("price >= ?", price).Order("price DESC, created_at ASC")
	}

	err := query.Find(&orders).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warnw("no matching orders found",
				"instrument_pair", instrumentPair,
				"order_type", orderType,
			)
			return nil, nil
		}
		r.log.Errorw("failed to get matching orders",
			"instrument_pair", instrumentPair,
			"order_type", orderType,
			"price", price,
			"is_buy_order", isBuyOrder,
			"error", err,
		)
		return nil, err
	}

	return orders, nil
}
