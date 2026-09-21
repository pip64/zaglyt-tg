package configs

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotName  string
	BotToken string

	Developers []int64

	IdeasChatID int64

	Driver   string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println(".env is not found")
	}

	devsStr := os.Getenv("DEVELOPERS")
	var developers []int64

	if devsStr != "" {
		fields := strings.Fields(devsStr)
		for _, field := range fields {
			id, err := strconv.ParseInt(field, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse developer ID %q: %w", field, err)
			}
			developers = append(developers, id)
		}
	}

	ideasChatIDStr := os.Getenv("IDEAS_CHAT")
	var ideasChatID int64

	if ideasChatIDStr != "" {
		id, err := strconv.ParseInt(ideasChatIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse developer ID %q: %w", ideasChatIDStr, err)
		}
		ideasChatID = id
	}

	cfg := &Config{
		BotName:     os.Getenv("BOT_NAME"),
		BotToken:    os.Getenv("BOT_TOKEN"),
		Developers:  developers,
		IdeasChatID: ideasChatID,
		Driver:      os.Getenv("DB_DRIVER"),
		Host:        os.Getenv("DB_HOST"),
		Port:        os.Getenv("DB_PORT"),
		User:        os.Getenv("DB_USER"),
		Password:    os.Getenv("DB_PASSWORD"),
		DBName:      os.Getenv("DB_NAME"),
	}

	if cfg.Host == "" || cfg.Port == "" || cfg.User == "" || cfg.DBName == "" {
		return nil, errors.New("some of DB_* values is empty")
	}

	if cfg.Driver == "" {
		cfg.Driver = "postgres"
	}

	return cfg, nil
}
