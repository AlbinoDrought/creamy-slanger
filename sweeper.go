package main

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

func sweepEvery(ctx context.Context, duration time.Duration) {
	var (
		sweeped uint64
		err     error
	)

	ticker := time.NewTicker(duration)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweeped, err = slangerOptions.EventManager.SweepExpired()
			if err != nil {
				log.WithField("error", err).Warn("sweep failed")
			}
			if sweeped > 0 {
				log.WithField("expired", sweeped).Debug("sweeped")
			}
		}
	}
}
