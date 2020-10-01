package playlist

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var (
	extm3u = []byte(`#EXTM3U`)
	ErrNoEXTM3U = errors.New(`No #EXTM3U found`)
	ErrNoSegments = errors.New(`No segments found`)
	RegexpVersion = regexp.MustCompile(`^#EXT-X-VERSION:(\d+)$`)
	RegexpTargetDuration = regexp.MustCompile(`^#EXT-X-TARGETDURATION:(\d+)$`)
	RegexpMediaSequence = regexp.MustCompile(`^#EXT-X-MEDIA-SEQUENCE:(\d+)$`)
	RegexpExtInf = regexp.MustCompile(`^#EXTINF:([\d\.]+),(.*)`)
	RegexpUri = regexp.MustCompile(`^([^\s#].*)`)
	RegexpEndList = regexp.MustCompile(`^#EXT-X-ENDLIST$`)
)

type Segment struct {
	Duration float32
	Title string
	Uri string
}

type Playlist struct {
	Version int
	TargetDuration int
	MediaSequence int
	EndList bool
	Segments []Segment
}


func readExtm3u(r io.Reader) error {
	p := make([]byte, len(extm3u))
	_, err := r.Read(p)
	if err != nil {
		log.Printf("Error during read #EXTM3U: %s\n", err.Error())
		return err
	}
	if !bytes.Equal(p, extm3u) {
		log.Printf("Got '%s' but expect '%s'\n", p, extm3u)
		return ErrNoEXTM3U
	}
	return nil
}

func parseInt(s string, r *regexp.Regexp, dst *int) bool {
	match := r.FindStringSubmatch(s)
	if len(match) != 2 { return false }
	i, err := strconv.ParseInt(match[1], 10, 32)
	if err != nil {
		log.Printf("Can't parse Int in '%s' with '%s': '%s'\n", s, r, err.Error())
		return false
	}
	*dst = int(i)
	return true
}

func parseExtinf(s string, dst **Segment) bool {
	match := RegexpExtInf.FindStringSubmatch(s)
	if len(match) != 3 { return false }
	i, err := strconv.ParseFloat(match[1], 32)
	if err != nil {
		log.Printf("Can't parse Float in '%s' with '%s': '%s'\n", s, RegexpExtInf, err.Error())
		return false
	}
	*dst = &Segment{
		Duration: float32(i),
		Title:    match[2],
	}
	return true
}

func parseUri(s string, dst *Segment) bool {
	if dst == nil { return false }
	match := RegexpUri.FindStringSubmatch(s)
	if len(match) != 2 {
		log.Printf("Segment '%v' found but can't parse uri '%s' with '%s'\n", *dst, s, RegexpUri)
		return false
	}
	dst.Uri = match[1]
	return true
}


func Parse(r io.Reader) (*Playlist, error) {
	if err := readExtm3u(r); err != nil { return nil, err }

	p := Playlist{
		Segments: make([]Segment, 0),
	}

	reader := bufio.NewReader(r)
	loop := true
	var segment *Segment
	for loop {
		line, err := reader.ReadString('\n')
		if err != nil {
			loop = false
			if err != io.EOF { log.Println(err.Error()) }
		}
		line = strings.TrimSuffix(line, "\n")
		if line == `` { continue }
		if parseExtinf(line, &segment) { continue }
		if parseUri(line, segment) {
			p.Segments = append(p.Segments, *segment)
			segment = nil
			continue
		}
		if parseInt(line, RegexpVersion, &p.Version) { continue }
		if parseInt(line, RegexpTargetDuration, &p.TargetDuration) { continue }
		if parseInt(line, RegexpMediaSequence, &p.MediaSequence) { continue }
		if RegexpEndList.MatchString(line) {
			p.EndList = true
			continue
		}
	}

	if len(p.Segments) == 0 {
		log.Printf("%v: %s\n", p, ErrNoSegments.Error())
		return nil, ErrNoSegments
	}
	return &p, nil
}

