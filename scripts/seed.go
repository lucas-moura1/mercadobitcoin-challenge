package main

import (
	"log"

	"github.com/google/uuid"
	"github.com/lucas-moura1/mercadobitcoin-challenge/config"
	"github.com/lucas-moura1/mercadobitcoin-challenge/entity"
	"github.com/shopspring/decimal"
)

func main() {
	db, err := config.SetupDatabase()
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	// Create accounts
	accounts := []entity.Account{
		{
			Base: entity.Base{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")},
			Name: "John Doe",
		},
		{
			Base: entity.Base{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222")},
			Name: "Jane Doe",
		},
	}

	for _, account := range accounts {
		if err := db.Create(&account).Error; err != nil {
			log.Printf("failed to create account %s: %v", account.Name, err)
			continue
		}
		log.Printf("created account: %s", account.Name)
	}

	// Create wallets
	wallets := []entity.Wallet{
		{
			Base:        entity.Base{ID: uuid.New()},
			AccountID:   accounts[0].ID,
			AssetSymbol: "BTC",
			Balance:     decimal.NewFromFloat(1.5),
		},
		{
			Base:        entity.Base{ID: uuid.New()},
			AccountID:   accounts[0].ID,
			AssetSymbol: "BRL",
			Balance:     decimal.NewFromFloat(200000.00),
		},
		{
			Base:        entity.Base{ID: uuid.New()},
			AccountID:   accounts[1].ID,
			AssetSymbol: "BTC",
			Balance:     decimal.NewFromFloat(0.5),
		},
		{
			Base:        entity.Base{ID: uuid.New()},
			AccountID:   accounts[1].ID,
			AssetSymbol: "BRL",
			Balance:     decimal.NewFromFloat(305000.00),
		},
	}

	for _, wallet := range wallets {
		if err := db.Create(&wallet).Error; err != nil {
			log.Printf("failed to create wallet for account %s: %v", wallet.AccountID, err)
			continue
		}
		log.Printf("created wallet: %s for account %s", wallet.AssetSymbol, wallet.AccountID)
	}

	log.Println("Seed completed successfully!")
}
