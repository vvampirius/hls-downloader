package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/vvampirius/hls-downloader/downloader"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const VERSION = `0.10`

var (
	ErrorLog = log.New(os.Stderr, `error#`, log.Lshortfile)
	DebugLog = log.New(os.Stdout, `debug#`, log.Lshortfile)
)

func helpText() {
	fmt.Println(`https://github.com/vvampirius/hls-downloader`)
	fmt.Println(`Download HTTP Live Streaming (HLS) content`)
	fmt.Printf("\nUsage: %s [options] [<m3u url> <output filename>]\n\n", os.Args[0])
	flag.PrintDefaults()
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
		if text == `` {
			continue
		}
		return text
	}
	return ``
}

func readOutputFilename() string {
	for {
		fmt.Print(`Filename: `)
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err.Error())
		}
		line = strings.TrimSuffix(line, "\n")
		if line == `` {
			return fmt.Sprintf("%d.mp4", time.Now().Unix())
		}
		if ext := filepath.Ext(line); ext == `` || len(ext) > 4 {
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

	var m3uUrl, outputFilename string
	switch flag.NArg() {
	case 2:
		m3uUrl, outputFilename = flag.Arg(0), flag.Arg(1)
		if _, err := os.Stat(outputFilename); err == nil {
			ErrorLog.Fatalln(`File exist!`)
		}
	case 0:
		outputFilename = readOutputFilename()
		m3uUrl = readUrl()
	default:
		os.Stdout = os.Stderr
		helpText()
		os.Exit(1)
	}

	useFfmpeg := true
	if *noffmpeg {
		useFfmpeg = false
	}

	notifyChan, err := downloader.NewDownloader().Download(m3uUrl, outputFilename, useFfmpeg, nil)
	if err != nil {
		ErrorLog.Println(err.Error())
		os.Exit(1)
	}

	i, segmentNum := 0, 0
	for d := range notifyChan {
		if i != 0 && segmentNum != d.CurrentSegment.Num {
			fmt.Println()
		}
		if d.Playlist != nil {
			fmt.Printf("\r[%d / %d] [%s / %s] [%.1f Mb] [%.1f / %.1f Kb]\t",
				d.CurrentSegment.Num, d.Playlist.SegmentsCount, time.Duration(d.DownloadedDuration*float32(time.Second)),
				time.Duration(d.Playlist.SegmentsDuration*float32(time.Second)), float64(d.GotBytes)/1024/1024,
				float32(d.CurrentSegment.GotBytes)/1024, float32(d.CurrentSegment.Size)/1024)
		} else {
			fmt.Printf("\rno playlist loaded")
		}
		i++
		segmentNum = d.CurrentSegment.Num
		err = d.Error
	}
	fmt.Println()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
