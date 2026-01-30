package telemetry

import (
	"context"
	"os"
	"time"

	"github.com/posthog/posthog-go"
	"go.uber.org/zap"
)

func SendStartupBeacon(component string, logger *zap.Logger) {
	if os.Getenv("ELASTI_TELEMETRY_ENABLED") == "false" {
		return
	}

	posthogKey := os.Getenv("POSTHOG_API_KEY")
	installID := os.Getenv("ELASTI_INSTALL_ID")

	if posthogKey == "" || installID == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sendBeacon(ctx, installID, component, posthogKey, logger)
	}()
}

func sendBeacon(_ context.Context, installID, component, posthogKey string, logger *zap.Logger) {
	endpoint := os.Getenv("POSTHOG_HOST")
	if endpoint == "" {
		endpoint = "https://app.posthog.com"
	}

	client, err := posthog.NewWithConfig(posthogKey, posthog.Config{
		Endpoint: endpoint,
		Interval: 1 * time.Second,
	})
	if err != nil {
		logger.Debug("Failed to init PostHog", zap.Error(err))
		return
	}
	defer client.Close()

	err = client.Enqueue(posthog.Capture{
		DistinctId: installID,
		Event:      "pod_started",
		Properties: map[string]interface{}{"component": component},
	})
	if err != nil {
		logger.Debug("Failed to send telemetry", zap.Error(err))
	}
}
