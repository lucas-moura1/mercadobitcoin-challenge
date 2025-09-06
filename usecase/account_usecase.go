package usecase

import (
	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/lucas-moura1/mercadobitcoin-challenge/repository"
	"go.uber.org/zap"
)

type accountUseCase struct {
	log              *zap.SugaredLogger
	walletRepository repository.WalletRepository
}

func NewAccountUseCase(
	log *zap.SugaredLogger,
	walletRepo repository.WalletRepository,
) AccountUseCase {
	return &accountUseCase{
		log:              log,
		walletRepository: walletRepo,
	}
}

func (u *accountUseCase) GetAccountBalance(accountID uuid.UUID) ([]*entity.Wallet, error) {
	u.log.Infow("fetching account balance", "account_id", accountID)

	wallets, err := u.walletRepository.GetByAccountID(accountID)
	if err != nil {
		return nil, err
	}

	if len(wallets) == 0 {
		return nil, nil
	}

	return wallets, nil
}
