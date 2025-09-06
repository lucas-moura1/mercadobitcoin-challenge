package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type walletRepository struct {
	log *zap.SugaredLogger
	db  *gorm.DB
}

func NewWalletRepository(log *zap.SugaredLogger, db *gorm.DB) WalletRepository {
	return &walletRepository{log: log, db: db}
}

func (r *walletRepository) chooseDB(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *walletRepository) Create(tx *gorm.DB, wallet *entity.Wallet) error {
	r.log.Debugw("creating wallet",
		"account_id", wallet.AccountID,
		"asset", wallet.AssetSymbol,
	)
	db := r.chooseDB(tx)

	if err := db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "account_id"}, {Name: "asset_symbol"}},
			DoNothing: true,
		}).Create(wallet).Error; err != nil {
		r.log.Errorw("failed to create wallet", "error", err)
		return err
	}

	return nil
}

func (r *walletRepository) GetByAccountID(accountID uuid.UUID) ([]*entity.Wallet, error) {
	var wallets []*entity.Wallet

	err := r.db.Where(&entity.Wallet{AccountID: accountID, DeletedAt: nil}).Find(&wallets).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warnw("no wallets found for account", "account_id", accountID)
			return nil, nil
		}
		r.log.Errorw("failed to get wallets", "account_id", accountID, "error", err)
		return nil, err
	}

	return wallets, nil
}

func (r *walletRepository) GetByAccountAndAsset(tx *gorm.DB, accountID uuid.UUID, assetSymbol string) (*entity.Wallet, error) {
	wallet := new(entity.Wallet)
	err := tx.Where("account_id = ? AND asset_symbol = ? AND deleted_at IS NULL", accountID, assetSymbol).
		First(wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warnw("no wallet found",
				"account_id", accountID,
				"asset", assetSymbol,
			)
			return nil, nil
		}
		r.log.Errorw("failed to get wallet",
			"account_id", accountID,
			"asset", assetSymbol,
			"error", err,
		)
		return nil, err
	}

	return wallet, nil
}

func (r *walletRepository) updateBalance(tx *gorm.DB, accountID uuid.UUID, assetSymbol string, amount decimal.Decimal, isAdd bool) error {
	r.log.Debugw("updating wallet balance", "account_id", accountID, "asset", assetSymbol, "amount", amount)
	updateClause := "balance - ?"
	if isAdd {
		updateClause = " balance + ?"
	}

	resp := tx.Model(&entity.Wallet{}).Where("account_id = ? AND asset_symbol = ? AND deleted_at IS NULL", accountID, assetSymbol).
		Update("balance", gorm.Expr(updateClause, amount))
	if resp.Error != nil {
		r.log.Errorw("failed to update wallet balance", "account_id", accountID, "asset", assetSymbol, "error", resp.Error)
		return resp.Error
	}
	if resp.RowsAffected == 0 {
		r.log.Warnw("no wallet found to update balance", "account_id", accountID, "asset", assetSymbol)
		return errors.New("insufficient balance or wallet not found")
	}

	return nil
}

func (r *walletRepository) SubtractFromBalance(tx *gorm.DB, accountID uuid.UUID, assetSymbol string, amount decimal.Decimal) error {
	r.log.Debugw("subtracting from wallet balance", "account_id", accountID, "asset", assetSymbol, "amount", amount)
	db := r.chooseDB(tx)
	if err := r.updateBalance(db, accountID, assetSymbol, amount, false); err != nil {
		return err
	}
	return nil
}

func (r *walletRepository) AddToBalance(tx *gorm.DB, accountID uuid.UUID, assetSymbol string, amount decimal.Decimal) error {
	r.log.Debugw("adding to wallet balance", "account_id", accountID, "asset", assetSymbol, "amount", amount)
	db := r.chooseDB(tx)
	if err := r.updateBalance(db, accountID, assetSymbol, amount, true); err != nil {
		return err
	}
	return nil
}
