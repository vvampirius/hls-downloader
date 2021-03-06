package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/vvampirius/hls-downloader/playlist"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const VERSION  = 0.8

func helpText() {
	fmt.Println(`https://github.com/vvampirius/hls-downloader`)
	fmt.Println(`Download HTTP Live Streaming (HLS) content`)
	fmt.Printf("\nUsage: %s [options] [<m3u url> <output filename>]\n\n", os.Args[0])
	flag.PrintDefaults()
}

func getBaseURL(playlistUrl string) (string, error) {
	u, err := url.Parse(playlistUrl)
	if err != nil {
		log.Println(err.Error())
		return ``, err
	}
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path.Dir(u.Path)), nil
}

func getChunksUrls(r io.Reader, baseUrl string) []string {
	chunksUrls := make([]string, 0)
	reader := bufio.NewReader(r)
	run := true
	for run {
		line, err := reader.ReadString('\n')
		if err != nil { run = false }
		if err != nil && err != io.EOF { log.Println(err.Error()) }
		if len(line) == 0 { continue }
		if string(line[0]) == `#` { continue }
		chunksUrls = append(chunksUrls, baseUrl+`/`+strings.TrimSuffix(line, "\n"))
	}
	return chunksUrls
}

func readChunk(chunkUrl string, w io.Writer) error {
	response, err := http.Get(chunkUrl)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return errors.New(fmt.Sprintf("%s %s", chunkUrl, response.Status))
	}

	if response.StatusCode != http.StatusOK {
		log.Println(response.Status)
	}

	n, err := io.Copy(w, response.Body)
	log.Printf("Got %d bytes\n", n)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}


func makeChunkUrl(baseUrl, segmentUri string) (string, error) {
	if baseUrl == `` || segmentUri == `` {
		err := errors.New(fmt.Sprintf("baseUrl: '%s', segmentUrl: '%s'", baseUrl, segmentUri))
		log.Println(err.Error())
		return ``, err
	}

	if string([]rune(segmentUri)[0]) != `/` { return baseUrl+`/`+segmentUri, nil }

	baseUrlParsed, err := url.Parse(baseUrl)
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	chunkUrl := fmt.Sprintf("%s://%s%s", baseUrlParsed.Scheme, baseUrlParsed.Host, segmentUri)
	return chunkUrl, nil
}

func download(playlistUrl string, w io.WriteCloser) error {
	baseUrl, err := getBaseURL(playlistUrl)
	if err != nil { return err }

	response, err := http.Get(playlistUrl)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer func() {
		if err := response.Body.Close(); err!=nil { log.Println(err.Error()) }
	}()

	p, err := playlist.Parse(response.Body)
	if err != nil { return err }
	log.Println(p) //TODO remove this

	for _, segment := range p.Segments {
		log.Println(segment)
		chunkUrl, err := makeChunkUrl(baseUrl, segment.Uri)
		if err != nil { return err }
		log.Println(chunkUrl)
		if err := readChunk(chunkUrl, w); err != nil {
			log.Println(err.Error())
			return err
		}
	}

	return nil
}

func readUrl() string {
	run := true
	for run {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter m3u URL: ")
		text, err := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		if err != nil {
			log.Println(err.Error())
			run = false
		}
		if text == `` { continue }
		return text
	}
	return ``
}

func readOutputFilename() string {
	for {
		fmt.Print(`Filename: `)
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil { log.Fatal(err.Error()) }
		line = strings.TrimSuffix(line, "\n")
		if line == `` { return fmt.Sprintf("%d.mp4", time.Now().Unix()) }
		if ext := filepath.Ext(line); ext==`` || len(ext) > 4 {
			log.Println(`Added mp4 extension`)
			line = line + `.mp4`
		}
		if _, err := os.Stat(line); err == nil {
			log.Printf("'%s' already exists\n", line)
			continue
		}
		return line
	}
}


func getFfmpegOutput(outputFilename string) (io.WriteCloser, error) {
	cmd := exec.Command(`ffmpeg`, `-f`, `mpegts`, `-vcodec`, `h264`, `-i`, `-`, `-codec`, `copy`, outputFilename)
	output, err := cmd.StdinPipe()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return output, nil
}


func getOutput(outputFilename string, useFfmpeg bool) (io.WriteCloser, error) {
	if useFfmpeg {
		if output, err := getFfmpegOutput(outputFilename); err == nil { return output, nil }
		log.Println(`Can't use ffmpeg! Trying to save to file as-is...`)
	}
	output, err := os.Create(outputFilename)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return output, nil
}


func main() {
	help := flag.Bool("h", false, "print this help")
	ver := flag.Bool("v", false, "Show version")
	noffmpeg := flag.Bool("noffmpeg", false, "Do not use ffmpeg")
	flag.Parse()

	if *help {
		helpText()
		os.Exit(0)
	}

	if *ver {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	log.SetFlags(log.Lshortfile)

	var m3uUrl, outputFilename string
	switch flag.NArg() {
	case 2:
		m3uUrl, outputFilename = flag.Arg(0), flag.Arg(1)
		if _, err := os.Stat(outputFilename); err == nil { log.Fatalln(`File exist!`) }
	case 0:
		outputFilename = readOutputFilename()
		m3uUrl = readUrl()
	default:
		os.Stdout = os.Stderr
		helpText()
		os.Exit(1)
	}

	useFfmpeg := true
	if *noffmpeg { useFfmpeg = false }
	output, err := getOutput(outputFilename, useFfmpeg)
	if err != nil { os.Exit(1) }

	if err := download(m3uUrl, output); err != nil {
		syscall.Exit(1)
	}
}
