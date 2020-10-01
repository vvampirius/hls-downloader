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
	"path"
	"strings"
	"syscall"
)

const VERSION  = 0.5

func helpText() {
	fmt.Println(`Download HTTP Live Streaming (HLS) content`)
	fmt.Printf("\nUsage: %s [options] [m3u url]\n\n", os.Args[0])
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

func main() {
	args := new(Args)
	help := flag.Bool("h", false, "print this help")
	ver := flag.Bool("v", false, "Show version")
	args.Output = flag.String(`o`, ``, `Output to <file>. Use '-' for stdout and empty for <unixtime>.mp4`)
	args.Overwrite = flag.String(`w`, `uniq`, `Overwrite output file (fail/uniq/overwrite)`)
	flag.Parse()

	if *help {
		helpText()
		os.Exit(0)
	}

	if *ver {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	if !args.CheckOverwrite() {
		fmt.Fprintf(os.Stderr, "Cant't use '%s' as overwite parameter!\n\n", *args.Overwrite)
		helpText()
		os.Exit(1)
	}

	log.SetFlags(log.Lshortfile)

	output, err := NewOutput(args)
	if err != nil { os.Exit(1) }
	log.Println(output)

	m3uUrl := ``
	if args := flag.Args(); len(args) > 0 { m3uUrl = args[0] }
	if m3uUrl == `` { m3uUrl = readUrl() }

	if err := download(m3uUrl, output); err != nil {
		syscall.Exit(1)
	}
}
