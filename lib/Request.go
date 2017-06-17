package telegram_youtube_bot

//go:generate stringer -type=DownloadStatus

type DownloadStatus int

const (
	WaitingForDownload DownloadStatus = iota
	Downloading
	DownloadDone
	Failed
	Stopped
	WaitingForUpload
	Uploading
	UploadDone
)

type DownloadResult struct {
	Id       int
	ChatId   int64
	Url      string
	FilePath string
	Status   DownloadStatus
	Req      *DownloadRequest
}

type DownloaderChannel chan chan *DownloadRequest
type UploaderChannel chan chan *DownloadResult
type DownloadResults []*DownloadResult

func NewDownloadResultPath(req *DownloadRequest, path string, status DownloadStatus) *DownloadResult {
	return &DownloadResult{
		Id:       req.id,
		ChatId:   req.chat_id,
		Url:      req.url,
		FilePath: path,
		Status:   status,
		Req:      req}
}

func NewDownloadResult(req *DownloadRequest, status DownloadStatus) *DownloadResult {
	return &DownloadResult{
		Id:       req.id,
		ChatId:   req.chat_id,
		Url:      req.url,
		FilePath: "",
		Status:   status,
		Req:      req}
}

type DownloadRequest struct {
	id         int
	chat_id    int64
	url        string
	stop       chan bool
	fileformat string
}

func NewDownloadRequest(id int, chat_id int64, url string) *DownloadRequest {
	return &DownloadRequest{
		id:         id,
		chat_id:    chat_id,
		url:        url,
		stop:       make(chan bool, 1),
		fileformat: "m4a"}
}

func (r *DownloadRequest) Stopped() bool {
	select {
	case <-r.stop:
		return true
	default:
		return false
	}
}

func (r *DownloadRequest) Stop() {
	select {
	case r.stop <- true:
	default:
	}
}
