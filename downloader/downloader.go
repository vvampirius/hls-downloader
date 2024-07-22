package downloader

import (
	"errors"
	"fmt"
	"github.com/vvampirius/hls-downloader/playlist"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Downloader struct {
	CurrentSegment struct {
		Num      int
		GotBytes int64
		Size     int64
		Url      string
	}
	DownloadedDuration float32
	Error              error
	Finished           bool
	GotBytes           int64
	Playlist           *playlist.Playlist
	Started            bool
}

func (downloader *Downloader) downloadChunk(notifyChan chan *Downloader, chunkUrl string, requestHeaders map[string]string, output io.Writer) error {
	request, err := http.NewRequest(http.MethodGet, chunkUrl, nil)
	if err != nil {
		ErrorLog.Println(err.Error())
		return err
	}
	if requestHeaders != nil {
		for k, v := range requestHeaders {
			request.Header.Set(k, v)
		}
	}
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		ErrorLog.Println(err)
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err := errors.New(fmt.Sprintf("%s %s", chunkUrl, response.Status))
		ErrorLog.Println(err.Error())
		return err
	}
	if contentLength := response.Header.Get(`Content-Length`); contentLength != `` {
		if n, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
			downloader.CurrentSegment.Size = n
		}
	}
	copyErrChan := make(chan error, 0)
	go func() {
		for {
			n, err := io.CopyN(output, response.Body, 1048576)
			downloader.CurrentSegment.GotBytes = downloader.CurrentSegment.GotBytes + n
			downloader.GotBytes = downloader.GotBytes + n
			copyErrChan <- err
			if err != nil {
				return
			}
		}
	}()
	for {
		select {
		case err := <-copyErrChan:
			if err == nil {
				notifyChan <- downloader
			} else if err == io.EOF {
				notifyChan <- downloader
				return nil
			} else {
				return err
			}
		case <-time.After(time.Second):
			notifyChan <- downloader
		}
	}
}

func (downloader *Downloader) downloadRoutine(notifyChan chan *Downloader, playlist *playlist.Playlist, output io.WriteCloser,
	baseUrl string, requestHeaders map[string]string) {
	defer close(notifyChan)
	defer output.Close()
	for {
		segment, err := playlist.GetSegment()
		if err != nil {
			downloader.Error = err
			notifyChan <- downloader
			return
		}
		if segment == nil {
			downloader.Finished = true
			notifyChan <- downloader
			return
		}
		downloader.CurrentSegment.Num++
		downloader.CurrentSegment.GotBytes = 0
		downloader.CurrentSegment.Url = segment.Uri
		notifyChan <- downloader
		chunkUrl, err := MakeChunkUrl(baseUrl, segment.Uri)
		if err != nil {
			downloader.Finished = true
			notifyChan <- downloader
			return
		}
		if err := downloader.downloadChunk(notifyChan, chunkUrl, requestHeaders, output); err != nil {
			downloader.Error = err
			notifyChan <- downloader
			return
		}
		downloader.DownloadedDuration = downloader.DownloadedDuration + segment.Duration
	}
}

func (downloader *Downloader) Download(playlistUrl, outputFilename string, useFfmpeg bool, requestHeaders map[string]string) (chan *Downloader, error) {
	if downloader.Started {
		err := errors.New(`already started`)
		ErrorLog.Println(err.Error())
		return nil, err
	}
	downloader.Started = true
	output, err := GetOutput(outputFilename, useFfmpeg)
	if err != nil {
		return nil, err
	}
	baseUrl, err := GetBaseURL(playlistUrl)
	if err != nil {
		return nil, err
	}
	playlist, err := GetPlaylistByUrl(playlistUrl, requestHeaders)
	if err != nil {
		return nil, err
	}
	downloader.Playlist = playlist
	notifyChan := make(chan *Downloader, 1)
	go downloader.downloadRoutine(notifyChan, playlist, output, baseUrl, requestHeaders)
	return notifyChan, nil
}

func NewDownloader() *Downloader {
	return &Downloader{}
}
