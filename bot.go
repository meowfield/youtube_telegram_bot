package main

import (
	"encoding/json"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"telegram_youtube_bot/lib"
)

const (
	ConfigFileName string = "config.json"
)

type BotConfig struct {
	Downloader   uint    `json:"downloaders"`
	Uploader     uint    `json:"uploaders"`
	Token        string  `json:"token"`
	APITimeout   int     `json:"api_timeout"`
	DebugBot     bool    `json:"debug_bot"`
	AllowedUsers []int64 `json:"allowed"`
}

func readConfig() (BotConfig, error) {
	conf := BotConfig{
		Downloader: 2,
		Uploader:   1,
		Token:      "",
		APITimeout: 60,
		DebugBot:   false}

	configFile, err := ioutil.ReadFile(ConfigFileName)
	if err != nil {
		return conf, err
	}

	if err := json.Unmarshal(configFile, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}

func main() {
	config, errConf := readConfig()
	if errConf != nil {
		log.Panic(errConf)
	}

	bot, errBot := tgbotapi.NewBotAPI(config.Token)
	if errBot != nil {
		log.Panic(errBot)
	}
	bot.Debug = config.DebugBot
	log.Printf("Authorized on account %s", bot.Self.UserName)

	dp := telegram_youtube_bot.NewDispatcher(config.Downloader, config.Uploader, bot, config.APITimeout, config.AllowedUsers)
	dp.Start()
}
