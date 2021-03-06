package telegram_youtube_bot

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
)

type Uploader struct {
	uploaderChannel UploaderChannel
	status          chan *DownloadResult
	bot             *tgbotapi.BotAPI
	requests        chan *DownloadResult
	quit            chan bool
}

func NewUploader(uploaders UploaderChannel, status chan *DownloadResult, bot *tgbotapi.BotAPI) *Uploader {
	return &Uploader{
		uploaderChannel: uploaders,
		status:          status,
		bot:             bot,
		requests:        make(chan *DownloadResult),
		quit:            make(chan bool)}
}

func (up *Uploader) Start() {
	go func() {
		for {
			up.uploaderChannel <- up.requests
			select {
			case <-up.quit:
				return
			case res := <-up.requests:
				up.status <- NewDownloadResult(res.Req, Uploading)
				log.Printf("Uploading %s to %d.", res.FilePath, res.ChatId)
				if _, err := os.Stat(res.FilePath); err == nil {
					msg := tgbotapi.NewAudioUpload(res.ChatId, res.FilePath)
					up.bot.Send(msg)
					up.status <- NewDownloadResult(res.Req, UploadDone)
				} else {
					log.Printf("Uploading %s to %d failed", res.FilePath, res.ChatId)
					up.status <- NewDownloadResult(res.Req, Failed)
				}
			}
		}
	}()
}

func (up *Uploader) Quit() {
	up.quit <- true
}
