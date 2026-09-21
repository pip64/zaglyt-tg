package handlers

import (
	"context"
	"zaglyt-tg/modules/helpers"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	goTelegramModels "github.com/go-telegram/bot/models"
)

func (h *Handler) StartCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		keyboard := helpers.GetLinksKeyboard()

		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text: `👋 Привет! Я - заглыт и работаю только в чатах!

В чатах ко мне нужно обращаться по имени или пинговать.`,
			ReplyMarkup: keyboard,
			ReplyParameters: &goTelegramModels.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
	}
}
