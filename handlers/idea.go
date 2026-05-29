package handlers

import (
	"context"
	"fmt"
	"strings"
	"zaglyt-tg/configs"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	goTelegramModels "github.com/go-telegram/bot/models"
)

func (h *Handler) IdeaCommandHandler(ctx context.Context, b *bot.Bot, update *goTelegramModels.Update) {
	if update.Message != nil {
		parts := strings.SplitN(update.Message.Text, " ", 2)
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Использование: `/idea <текст идеи>`",
				ReplyParameters: &goTelegramModels.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})
			return
		}

		ideaText := strings.TrimSpace(parts[1])

		config, err := configs.LoadConfig()
		if err != nil {
			fmt.Println(err)
			return
		}

		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: config.IdeasChatID,
			Text:   ideaText,
		})

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      `Идея успешно отправлена и появится в опросе за идеи в [канале](https://t.me/zaglit) заглыта уже в скором времени\!`,
			ParseMode: models.ParseModeMarkdown,
			ReplyParameters: &goTelegramModels.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
