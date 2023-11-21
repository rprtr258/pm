package main

import (
	"fmt"

	"github.com/rprtr258/pm/internal/infra/linuxprocess"
	"github.com/rprtr258/pm/internal/infra/log"
)

func main() {
	status, err := linuxprocess.GetSelfStatus()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get self status: %w", err))
	}

	log.Info().Any("status", status).Send()
}
