package v1

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
)

// State is per report chunk: retries skip confirmed deliveries and release only the failed chunk.
type telegramReportPartDelivery interface {
	Claim(context.Context, int) (bool, error)
	Send(context.Context, string) error
	Release(context.Context, int) error
	MarkSent(context.Context, int) error
}

func deliverTelegramReportParts(ctx context.Context, messages []string, delivery telegramReportPartDelivery) error {
	for index, message := range messages {
		claimed, err := delivery.Claim(ctx, index)
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}
		if err := delivery.Send(ctx, message); err != nil {
			// A transport timeout may occur after Telegram accepted the message. Keep its claim rather than resend uncertain delivery.
			var rejected *models.TelegramAPIRejectedError
			if errors.As(err, &rejected) {
				if releaseErr := delivery.Release(ctx, index); releaseErr != nil {
					return releaseErr
				}
			}
			return err
		}
		if err := delivery.MarkSent(ctx, index); err != nil {
			return err
		}
	}
	return nil
}
