package downloader

import (
	"errors"
	"fmt"
	"github.com/vvampirius/hls-downloader/playlist"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
)

var (
	ErrorLog = log.New(os.Stderr, `error#`, log.Lshortfile)
	DebugLog = log.New(os.Stdout, `debug#`, log.Lshortfile)
)

func GetBaseURL(playlistUrl string) (string, error) {
	u, err := url.Parse(playlistUrl)
	if err != nil {
		ErrorLog.Println(err.Error())
		return ``, err
	}
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path.Dir(u.Path)), nil
}

func GetFfmpegOutput(outputFilename string) (io.WriteCloser, error) {
	cmd := exec.Command(`ffmpeg`, `-f`, `mpegts`, `-vcodec`, `h264`, `-i`, `-`, `-codec`, `copy`, outputFilename)
	output, err := cmd.StdinPipe()
	if err != nil {
		ErrorLog.Println(err.Error())
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		ErrorLog.Println(err.Error())
		return nil, err
	}
	return output, nil
}

func GetOutput(outputFilename string, useFfmpeg bool) (io.WriteCloser, error) {
	if useFfmpeg {
		output, err := GetFfmpegOutput(outputFilename)
		if err == nil {
			return output, nil
		} else {
			ErrorLog.Println(err.Error())
			ErrorLog.Println(`Can't use ffmpeg! Trying to save to file as-is...`)
		}
	}
	output, err := os.Create(outputFilename)
	if err != nil {
		ErrorLog.Println(err.Error())
		return nil, err
	}
	return output, nil
}

func GetPlaylistByUrl(playlistUrl string, requestHeaders map[string]string) (*playlist.Playlist, error) {
	request, err := http.NewRequest(http.MethodGet, playlistUrl, nil)
	if err != nil {
		ErrorLog.Println(err.Error())
		return nil, err
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
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		DebugLog.Println(playlistUrl, response.Status)
	}
	return playlist.Parse(response.Body), nil
}

func MakeChunkUrl(baseUrl, segmentUri string) (string, error) {
	if baseUrl == `` || segmentUri == `` {
		err := errors.New(fmt.Sprintf("baseUrl: '%s', segmentUrl: '%s'", baseUrl, segmentUri))
		ErrorLog.Println(err.Error())
		return ``, err
	}

	if string([]rune(segmentUri)[0]) != `/` {
		return baseUrl + `/` + segmentUri, nil
	}

	baseUrlParsed, err := url.Parse(baseUrl)
	if err != nil {
		ErrorLog.Println(err.Error())
		return "", err
	}
	chunkUrl := fmt.Sprintf("%s://%s%s", baseUrlParsed.Scheme, baseUrlParsed.Host, segmentUri)
	return chunkUrl, nil
}
