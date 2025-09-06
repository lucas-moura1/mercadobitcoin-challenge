package repository

import (
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type accountRepository struct {
	log *zap.SugaredLogger
	db  *gorm.DB
}

func NewAccountRepository(log *zap.SugaredLogger, db *gorm.DB) AccountRepository {
	return &accountRepository{log: log, db: db}
}

func (r *accountRepository) Create(account *entity.Account) error {
	r.log.Debugw("creating account", "name", account.Name)

	if err := r.db.Create(account).Error; err != nil {
		r.log.Errorw("failed to create account", "error", err)
		return err
	}

	return nil
}
