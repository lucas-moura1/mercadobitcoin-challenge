package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/usecase"
	"go.uber.org/zap"
)

type accountHandler struct {
	log            *zap.SugaredLogger
	accountUseCase usecase.AccountUseCase
}

func NewAccountHandler(log *zap.SugaredLogger, accountUseCase usecase.AccountUseCase) *accountHandler {
	return &accountHandler{log: log, accountUseCase: accountUseCase}
}

type GetAccountBalanceResponse struct {
	AccountID uuid.UUID       `json:"account_id"`
	Balances  []*AssetBalance `json:"balances"`
}

type AssetBalance struct {
	Asset   string `json:"asset"`
	Balance string `json:"balance"`
}

func (h *accountHandler) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.log.Errorw("invalid account id", "error", err)
		errorHandler(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	h.log.Infow("getting account balance", "account_id", accountID)

	wallets, err := h.accountUseCase.GetAccountBalance(accountID)
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, err.Error())
		return
	}

	if wallets == nil {
		errorHandler(w, http.StatusNotFound, "No wallets found")
		return
	}

	balances := make([]*AssetBalance, len(wallets))
	for i, wallet := range wallets {
		balances[i] = &AssetBalance{
			Asset:   wallet.AssetSymbol,
			Balance: wallet.Balance.String(),
		}
	}

	response := GetAccountBalanceResponse{
		AccountID: accountID,
		Balances:  balances,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
