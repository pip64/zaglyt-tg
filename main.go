package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"zaglyt-tg/app"
	"zaglyt-tg/configs"
	"zaglyt-tg/handlers"
	"zaglyt-tg/middlewares"
	"zaglyt-tg/repository"
	"zaglyt-tg/repository/channel"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	cfg, err := configs.LoadConfig()
	if err != nil {
		panic(err)
	}

	db, err := repository.InitDB(cfg)
	if err != nil {
		log.Fatalf("db initialization error: %v", err)
	}
	defer db.Close()

	channelRepo := channel.NewChannelRepository(db)

	app := app.NewApp(channelRepo)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var handler *handlers.Handler

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if handler != nil {
				handler.MessageHandler(ctx, b, update)
			}
		}),
		bot.WithMiddlewares(middlewares.RecoveryMiddleware),
	}

	b, err := bot.New(cfg.BotToken, opts...)
	if err != nil {
		panic(err)
	}

	bot_info, err := b.GetMe(ctx)
	if err != nil {
		log.Fatalf("failed to get bot info: %v", err)
	}

	h := handlers.NewHandler(app, bot_info)
	handler = &h

	//commands
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, handler.StartCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/switcher", bot.MatchTypePrefix, handler.SwitcherCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/clear", bot.MatchTypePrefix, handler.ClearCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/download", bot.MatchTypePrefix, handler.DownloadCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/anecdote", bot.MatchTypePrefix, handler.GenerateAnecdoteCommandHandler)

	//admin commands
	b.RegisterHandler(bot.HandlerTypeMessageText, "/whoami", bot.MatchTypeExact, handler.WhoAmICommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/chatid", bot.MatchTypeExact, handler.GetChatIDCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stats", bot.MatchTypeExact, handler.GetBotStatsCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/broadcast", bot.MatchTypePrefix, handler.BroadcastCommandHandler)

	// callbacks
	b.RegisterHandler(
		bot.HandlerTypeCallbackQueryData,
		"bot_clear",
		bot.MatchTypePrefix,
		handler.CallbackClear,
	)
	b.RegisterHandler(
		bot.HandlerTypeCallbackQueryData,
		"bot_",
		bot.MatchTypePrefix,
		handler.CallbackBotSwitcher,
	)

	b.Start(ctx)
}
