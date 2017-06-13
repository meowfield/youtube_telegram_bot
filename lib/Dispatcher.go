package telegram_youtube_bot

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
)

const (
	NoActiveJob    string = "There are currently no active downloads or uploads."
	StoppedJobs    string = "Stopped %d job(s)"
	State          string = "%s state: %s"
	DownloadFailed string = "Download %s failed!"
)

type Dispatcher struct {
	done           chan *DownloadResult
	downloaders    DownloaderChannel
	uploaders      UploaderChannel
	bot            *tgbotapi.BotAPI
	apitimeout     int
	numDownloaders uint
	numUploaders   uint
	results        map[int64]DownloadResults
}

func NewDispatcher(numDownloaders, numUploaders uint, bot *tgbotapi.BotAPI, timeout int) *Dispatcher {
	r := &Dispatcher{
		done:           make(chan *DownloadResult),
		downloaders:    make(chan chan *DownloadRequest),
		uploaders:      make(chan chan *DownloadResult),
		bot:            bot,
		apitimeout:     timeout,
		numDownloaders: numDownloaders,
		numUploaders:   numUploaders,
		results:        make(map[int64]DownloadResults)}
	return r
}

func (d *Dispatcher) Start() {
	d.dispatch()
}

func (d *Dispatcher) startWorkers() {
	for dl := uint(0); dl < d.numDownloaders; dl++ {
		downloader := NewDownloader(d.downloaders, d.done)
		downloader.Start()
	}
	for up := uint(0); up < d.numUploaders; up++ {
		uploader := NewUploader(d.uploaders, d.done, d.bot)
		uploader.Start()
	}
}

func (d *Dispatcher) dispatch() {
	download_id := 0
	u := tgbotapi.NewUpdate(0)
	u.Timeout = d.apitimeout
	updates, errChan := d.bot.GetUpdatesChan(u)

	if errChan != nil {
		log.Panic(errChan)
	}

	d.startWorkers()

	for {
		select {
		case update := <-updates:
			if update.Message != nil {
				msg := update.Message
				log.Printf("User: %d %s\n", msg.From.ID, msg.From.FirstName)
				switch msg.Text {
				case "/start":

				case "/stop":
					d.handleStopMsg(msg.Chat.ID)
				case "/status":
					d.handleStatusMsg(msg.Chat.ID)
				default:
					download_id = download_id + 1
					req := NewDownloadRequest(download_id, msg.Chat.ID, msg.Text)
					d.results[msg.Chat.ID] = append(d.results[msg.Chat.ID], NewDownloadResult(req, WaitingForDownload))
					go func(r *DownloadRequest) {
						worker := <-d.downloaders
						worker <- r
					}(req)
				}
			}
		case result := <-d.done:
			log.Printf("Request-Result for DL-ID: %d URL: %s Status: %s\n", result.Id, result.Url, result.Status)
			d.handleResult(result)
		}
	}
}

func (d *Dispatcher) handleStopMsg(chatID int64) {
	cnt := len(d.results[chatID])
	if cnt == 0 {
		d.bot.Send(tgbotapi.NewMessage(chatID, NoActiveJob))
		return
	}
	for _, result := range d.results[chatID] {
		result.Req.Stop()
	}
	reply := tgbotapi.NewMessage(chatID, fmt.Sprintf(StoppedJobs, cnt))
	d.bot.Send(reply)
}

func (d *Dispatcher) handleStatusMsg(chatID int64) {
	if len(d.results[chatID]) == 0 {
		d.bot.Send(tgbotapi.NewMessage(chatID, NoActiveJob))
		return
	}

	for _, result := range d.results[chatID] {
		reply := tgbotapi.NewMessage(chatID, fmt.Sprintf(State, result.Url, result.Status))
		reply.DisableWebPagePreview = true
		d.bot.Send(reply)
	}
}

func (d *Dispatcher) handleResult(result *DownloadResult) {
	if _, found := d.results[result.ChatId]; !found {
		log.Printf("Worker returned result for unknown chatid: %d", result.ChatId)
		return
	}

	for i, r := range d.results[result.ChatId] {
		if result.Id == r.Id {
			d.results[result.ChatId][i] = result

			switch result.Status {
			case DownloadDone:
				d.results[result.ChatId][i].Status = WaitingForUpload
				go func(r *DownloadResult) {
					uploader := <-d.uploaders
					uploader <- r
				}(result)
			case Failed:
				failedMsg := tgbotapi.NewMessage(result.ChatId, fmt.Sprintf(DownloadFailed, result.Url))
				d.bot.Send(failedMsg)
				fallthrough
			case Stopped, UploadDone:
				d.results[result.ChatId] = append(d.results[result.ChatId][:i], d.results[result.ChatId][i+1:]...)
			}
		}
	}
}
