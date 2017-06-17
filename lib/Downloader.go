package telegram_youtube_bot

import (
	_ "bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	_ "strings"
	_ "time"
)

const (
	YoutubeDLCmd = "%s --prefer-ffmpeg --add-metadata --print-json --audio-format m4a -x \"%s\""
)

//go:generate stringer -type=YoutubeInfo
type YoutubeInfo struct {
	Filesize    int    `json:"filesize"`
	UploaderID  string `json:"uploader_id"`
	URL         string `json:"url"`
	Filename    string `json:"_filename"`
	Creator     string `json:"creator"`
	WebpageURL  string `json:"webpage_url"`
	Uploader    string `json:"uploader"`
	Fulltitle   string `json:"fulltitle"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	UploadDate  string `json:"upload_date"`
	Description string `json:"description"`
	err         error
}

type Downloader struct {
	downloaderChannel DownloaderChannel
	status            chan *DownloadResult
	requests          chan *DownloadRequest
	quit              chan bool
}

func NewDownloader(downloaderChannel DownloaderChannel, status chan *DownloadResult) *Downloader {
	return &Downloader{
		downloaderChannel: downloaderChannel,
		status:            status,
		requests:          make(chan *DownloadRequest),
		quit:              make(chan bool)}
}

func (dl *Downloader) Start() {
	go func() {
		for {
			dl.downloaderChannel <- dl.requests
			select {
			case <-dl.quit:
				return
			case req := <-dl.requests:
				if req.Stopped() {
					dl.status <- NewDownloadResult(req, Stopped)
				} else {
					dl.status <- NewDownloadResult(req, Downloading)
					dl.download(req)
				}
			}
		}
	}()
}
func (dl *Downloader) download(req *DownloadRequest) {
	op := fmt.Sprintf(YoutubeDLCmd, "/usr/local/bin/youtube-dl", req.url)

	done := make(chan YoutubeInfo, 1)
	cmd := exec.Command("sh", "-c", op)
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	go func() {
		yt := YoutubeInfo{}
		json.NewDecoder(stdout).Decode(&yt)
		yt.err = cmd.Wait()
		done <- yt
	}()

	select {
	case yt := <-done:
		if yt.err != nil {
			log.Println(yt.err)
			dl.status <- NewDownloadResult(req, Failed)
		} else {
			dl.status <- NewDownloadResultPath(req, yt.Filename, DownloadDone)
		}
	case <-req.stop:
		cmd.Process.Kill()
		dl.status <- NewDownloadResult(req, Stopped)
	}
}

func (dl *Downloader) Stop() {
	dl.quit <- true
}
