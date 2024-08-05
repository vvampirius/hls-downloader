package main

import (
	"fmt"
	"io"
	"net/http"
)

type EventStream struct {
	w        io.Writer
	Error    error
	DataChan chan string
}

func (es *EventStream) Stream() {
	for data := range es.DataChan {
		if _, err := fmt.Fprintf(es.w, "data: %s\n\n", data); err != nil {
			ErrorLog.Println(err.Error())
			es.Error = err
			close(es.DataChan)
		}
		es.w.(http.Flusher).Flush()
	}
	es.DataChan = nil
}

func NewEventStream(w http.ResponseWriter) *EventStream {
	es := EventStream{
		w:        w,
		DataChan: make(chan string, 1),
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	return &es
}
