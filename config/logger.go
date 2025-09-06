package config

import (
	"os"

	"go.uber.org/zap"
)

func SetupLogger() (*zap.SugaredLogger, error) {
	var logger *zap.Logger
	var err error

	if os.Getenv("ENV") == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		return nil, err
	}
	defer logger.Sync()

	return logger.Sugar(), nil
}
