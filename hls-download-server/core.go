package main

import (
	"errors"
	"fmt"
	"github.com/vvampirius/hls-downloader/downloader"
	"net/http"
	"strconv"
	"time"
)

type Core struct {
	Tasks []*Task
}

func (core *Core) addHandler(w http.ResponseWriter, r *http.Request) {
	DebugLog.Printf("%s %s %s '%s'", r.Header.Get(`X-Real-IP`), r.Method, r.RequestURI, r.UserAgent())
	taskUrl := r.URL.Query().Get(`url`)
	if taskUrl == `` {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, `URL is empty`)
		return
	}
	filename := r.URL.Query().Get(`filename`)
	if filename == `` {
		filename = fmt.Sprintf(`%d.mp4`, time.Now().Unix())
	}
	task := Task{
		EventStreams: make([]*EventStream, 0),
		Url:          taskUrl,
		Filename:     filename,
		Downloader:   downloader.NewDownloader(),
	}
	if source := r.URL.Query().Get(`source`); source != `` {
		task.Source = source
	}
	requestHeaders := make(map[string]string)
	if referer := r.Header.Get(`Referer`); referer != `` && r.Header.Get(`ignore_referrer`) != `true` {
		requestHeaders[`Referer`] = referer
	}
	if userAgent := r.Header.Get(`User-Agent`); userAgent != `` {
		requestHeaders[`User-Agent`] = userAgent
	}
	c, err := task.Downloader.Download(taskUrl, filename, true, requestHeaders)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
	core.Tasks = append(core.Tasks, &task)
	go func() {
		for range c {
			//DebugLog.Println(d)
			for _, es := range task.EventStreams {
				//DebugLog.Println(es)
				if es.DataChan != nil && es.Error == nil {
					ti := task.GetInfo()
					es.DataChan <- ti.Json()
				}
			}
		}
		task.Finished = true
	}()
	http.Redirect(w, r, fmt.Sprintf(`/%d/`, len(core.Tasks)-1), http.StatusFound)
}

func (core *Core) getTask(id string) (*Task, error) {
	n, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		ErrorLog.Println(id, err.Error())
		return nil, err
	}
	if len(core.Tasks)-1 < int(n) {
		err = errors.New(fmt.Sprintf("%d not found", n))
		ErrorLog.Println(err.Error())
		return nil, err
	}
	return core.Tasks[int(n)], nil
}

func (core *Core) indexHandler(w http.ResponseWriter, r *http.Request) {
	DebugLog.Printf("%s %s %s '%s'", r.Header.Get(`X-Real-IP`), r.Method, r.RequestURI, r.UserAgent())
	t := getTemplate(`index.html`, indexTemplate)
	if err := t.Execute(w, core.Tasks); err != nil {
		ErrorLog.Println(err.Error())
	}
}

func (core *Core) taskHandler(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get(`Accept`)
	DebugLog.Printf("%s %s %s '%s' '%s'", r.Header.Get(`X-Real-IP`), r.Method, r.RequestURI, r.UserAgent(),
		accept)
	task, err := core.getTask(r.PathValue(`task`))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, err.Error())
		return
	}
	switch accept {
	case `text/event-stream`:
		es := NewEventStream(w)
		ti := task.GetInfo()
		es.DataChan <- ti.Json()
		task.EventStreams = append(task.EventStreams, es)
		es.Stream()
		break

	default:
		t := getTemplate(`task.html`, taskTemplate)
		var data struct {
			TaskId string
			Task   *Task
		}
		data.TaskId = r.PathValue(`task`)
		data.Task = task
		if err := t.Execute(w, data); err != nil {
			ErrorLog.Println(err.Error())
		}
	}
}

func NewCore() *Core {
	core := Core{
		Tasks: make([]*Task, 0),
	}
	return &core
}
