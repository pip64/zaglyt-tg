package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"zaglyt-tg/modules/helpers"
	"zaglyt-tg/modules/messages"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) GenerateAnecdoteCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		channel, err := h.app.GetChatByID(ctx, update.Message.Chat.ID)
		if err != nil {
			fmt.Println(err)
			return
		}

		var fileName string
		if update.Message.Chat.Type == "private" {
			fileName = "dm"
		} else {
			fileName = strconv.FormatInt(channel.ChannelID, 10)
		}

		db, err := messages.Read(fileName)
		if err != nil {
			fmt.Println(err)
			return
		}

		dataset := strings.Split(db, "\n")
		if len(dataset) == 0 {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   `История сообщений этого чата пуста.`,
				ReplyParameters: &models.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})

			return
		}

		anecdote, err := helpers.GenerateAnecdote(dataset)
		if err != nil {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   `Не удалось сгенерировать анекдот.`,
				ReplyParameters: &models.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})

			return
		}

		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   anecdote,
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
	}
}
