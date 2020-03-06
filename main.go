package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"syscall"
)

const VERSION  = 0.1

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
		return errors.New(`not found`)
	}

	log.Println(response.StatusCode)

	n, err := io.Copy(w, response.Body)
	log.Printf("Got %d bytes\n", n)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func download(playlistUrl string, filePath string) error {
	baseUrl, err := getBaseURL(playlistUrl)
	if err != nil { return err }

	response, err := http.Get(playlistUrl)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	chunksUrls := getChunksUrls(response.Body, baseUrl)
	if err := response.Body.Close(); err!=nil { log.Println(err.Error()) }

	w, err := os.Create(filePath)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer w.Close()

	for _, chunksUrl := range chunksUrls {
		log.Println(chunksUrl)
		if err := readChunk(chunksUrl, w); err != nil {
			log.Println(err.Error())
			return err
		}
	}

	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)

	log.Println(os.Args)

	if err := download(os.Args[1], os.Args[2]); err != nil {
		syscall.Exit(1)
	}
}
