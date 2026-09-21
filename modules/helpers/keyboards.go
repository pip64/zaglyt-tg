package helpers

import (
	"github.com/go-telegram/bot/models"
)

func GetSwitcherKeyboard(enabled bool) *models.InlineKeyboardMarkup {
	textEnable := "Включить"
	textDisable := "• Выключен •"

	if enabled {
		textEnable = "• Включен •"
		textDisable = "Выключить"
	}

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         textEnable,
					CallbackData: "bot_enable",
				},
				{
					Text:         textDisable,
					CallbackData: "bot_disable",
				},
			},
		},
	}
}

func GetClearKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "Я уверен.",
					CallbackData: "bot_clear",
				},
			},
		},
	}
}

func GetLinksKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text: "Добавить в чат",
					URL:  "https://t.me/zaglit_bot?startgroup=start",
				},
			},
			{
				{
					Text: "Канал Заглыта",
					URL:  "https://t.me/zaglit",
				},
			},
		},
	}
}
