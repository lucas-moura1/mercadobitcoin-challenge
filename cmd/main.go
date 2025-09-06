package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucas-moura1/mercadobitcoin-challenge/config"
	"github.com/lucas-moura1/mercadobitcoin-challenge/handler"
	"github.com/lucas-moura1/mercadobitcoin-challenge/repository"
	"github.com/lucas-moura1/mercadobitcoin-challenge/usecase"
)

func main() {
	log, err := config.SetupLogger()
	if err != nil {
		panic(err)
	}

	db, err := config.SetupDatabase()
	if err != nil {
		panic(err)
	}

	orderRepository := repository.NewOrderRepository(log, db)
	walletRepository := repository.NewWalletRepository(log, db)
	tradeRepository := repository.NewTradeRepository(log)

	orderUsecase := usecase.NewOrderUseCase(log, orderRepository, walletRepository, tradeRepository, db)
	accountUsecase := usecase.NewAccountUseCase(log, walletRepository)

	orderHandler := handler.NewOrderHandler(log, orderUsecase)
	accountHandler := handler.NewAccountHandler(log, accountUsecase)

	http.HandleFunc("POST /orders", orderHandler.CreateOrder)
	http.HandleFunc("POST /orders/{id}/cancel", orderHandler.CancelOrder)
	http.HandleFunc("GET /orders/{instrument_pair}", orderHandler.GetOrderBook)

	http.HandleFunc("GET /accounts/{id}/balance", accountHandler.GetAccountBalance)

	server := &http.Server{Addr: fmt.Sprintf(":%s", os.Getenv("PORT"))}

	go func() {
		log.Info("Server started at :8080")
		if err := server.ListenAndServe(); err != nil && http.ErrServerClosed != err {
			panic(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		panic(err)
	}
	log.Info("Server gracefully stopped!")
}
