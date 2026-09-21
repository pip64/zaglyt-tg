// admin commands
package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"zaglyt-tg/modules/helpers"

	"github.com/go-telegram/bot"
	goTelegramModels "github.com/go-telegram/bot/models"
)

func (h *Handler) WhoAmICommandHandler(ctx context.Context, b *bot.Bot, update *goTelegramModels.Update) {
	if update.Message != nil {
		if helpers.IsUserDeveloper(update.Message.From.ID) {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Разработчик.",
				ReplyParameters: &goTelegramModels.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})
		}
	}
}

func (h *Handler) GetChatIDCommandHandler(ctx context.Context, b *bot.Bot, update *goTelegramModels.Update) {
	if update.Message != nil {
		if helpers.IsUserDeveloper(update.Message.From.ID) {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   strconv.FormatInt(update.Message.Chat.ID, 10),
				ReplyParameters: &goTelegramModels.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})
		}
	}
}

func (h *Handler) GetBotStatsCommandHandler(ctx context.Context, b *bot.Bot, update *goTelegramModels.Update) {
	if update.Message != nil {
		if helpers.IsUserDeveloper(update.Message.From.ID) {
			stats, err := h.app.GetBotStats(ctx)
			if err != nil {
				fmt.Println(err)
				return
			}

			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   fmt.Sprintf("1. Чатов в БД: %d\n2. Чатов в БД, где бот включен: %d\n3. Чатов в БД, где бот выключен: %d", stats.TotalChats, stats.EnabledChats, stats.TotalChats-stats.EnabledChats),
				ReplyParameters: &goTelegramModels.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})
		}
	}
}

func (h *Handler) BroadcastCommandHandler(ctx context.Context, b *bot.Bot, update *goTelegramModels.Update) {
	if update.Message == nil {
		return
	}

	if !helpers.IsUserDeveloper(update.Message.From.ID) {
		return
	}

	parts := strings.SplitN(update.Message.Text, " ", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Использование: `/broadcast <текст сообщения>`",
			ReplyParameters: &goTelegramModels.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
		return
	}

	broadcastText := strings.TrimSpace(parts[1])

	chatIDs, err := h.app.GetActiveChats(ctx)
	if err != nil {
		fmt.Println(err)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Не удалось получить список чатов для рассылки.",
		})
		return
	}

	totalChats := len(chatIDs)
	if totalChats == 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Список активных чатов пуст.",
		})
		return
	}

	statusMessage, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("Рассылка запущена фоном для %d чатов.", totalChats),
	})

	go h.runBackgroundBroadcast(context.Background(), b, update.Message.Chat.ID, statusMessage.ID, chatIDs, broadcastText)
}

func (h *Handler) runBackgroundBroadcast(ctx context.Context, b *bot.Bot, adminChatID int64, statusMsgID int, chatIDs []int64, text string) {
	successCount := 0
	failCount := 0

	ticker := time.NewTicker(35 * time.Millisecond)
	defer ticker.Stop()

	for _, chatID := range chatIDs {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   text,
			})
			if err != nil {
				failCount++
			} else {
				successCount++
			}
		}
	}

	reportText := fmt.Sprintf("Фоновая рассылка завершена.\n\nВсего чатов: %d\nУспешно: %d\nОшибок: %d", len(chatIDs), successCount, failCount)
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: adminChatID,
		Text:   reportText,
		ReplyParameters: &goTelegramModels.ReplyParameters{
			MessageID: statusMsgID,
		},
	})
}
