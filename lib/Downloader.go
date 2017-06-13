package telegram_youtube_bot

import (
	"fmt"
	"os/exec"
	"time"
)

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
	path := fmt.Sprintf("%d-%d", req.chat_id, time.Now().UnixNano())
	ext := "m4a"
	op := fmt.Sprintf("%s --prefer-ffmpeg -x --audio-format %s -o \"%s.%%(ext)s\" \"%s\"",
		"/usr/local/bin/youtube-dl",
		ext,
		path,
		req.url)
	done := make(chan error, 1)
	cmd := exec.Command("sh", "-c", op)
	cmd.Start()
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			dl.status <- NewDownloadResult(req, Failed)
		} else {
			path_with_ext := path + "." + ext
			dl.status <- NewDownloadResultPath(req, path_with_ext, DownloadDone)
		}
	case <-req.stop:
		cmd.Process.Kill()
		dl.status <- NewDownloadResult(req, Stopped)
	}
}

func (dl *Downloader) Stop() {
	dl.quit <- true
}
