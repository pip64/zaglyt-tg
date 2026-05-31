package helpers

import (
	"context"

	"github.com/go-telegram/bot"
)

func IsUserAdmin(ctx context.Context, b *bot.Bot, chatID int64, userID int64, chatType string) (bool, error) {
	if chatType == "private" {
		return true, nil
	}

	member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{
		ChatID: chatID,
		UserID: userID,
	})
	if err != nil {
		return false, err
	}

	if member.Owner != nil {
		return true, nil
	}

	if member.Administrator != nil {
		return true, nil
	}

	return false, nil
}
