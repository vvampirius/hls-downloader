package main

import (
	"encoding/json"
	"github.com/vvampirius/hls-downloader/downloader"
)

type TaskInfo struct {
	Filename       string `json:"filename"`
	Url            string `json:"url"`
	CurrentSegment struct {
		Num      int    `json:"num"`
		GotBytes int64  `json:"got_bytes"`
		Size     int64  `json:"size"`
		Url      string `json:"url"`
	} `json:"current_segment"`
	DownloadedDuration float32 `json:"downloaded_duration"`
	Error              string  `json:"error"`
	Finished           bool    `json:"finished"`
	GotBytes           int64   `json:"got_bytes"`
	Started            bool    `json:"started"`
	SegmentsCount      int     `json:"segments_count"`
	SegmentsDuration   float32 `json:"segments_duration"`
	Source             string
}

func (ti *TaskInfo) Json() string {
	data, err := json.Marshal(*ti)
	if err != nil {
		ErrorLog.Println(err.Error())
	}
	return string(data)
}

type Task struct {
	EventStreams []*EventStream
	Filename     string
	Url          string
	Downloader   *downloader.Downloader
	Finished     bool
	Source       string
}

func (task *Task) GetInfo() TaskInfo {
	ti := TaskInfo{
		Filename: task.Filename,
		Url:      task.Url,
		Source:   task.Source,
	}
	if task.Downloader != nil {
		ti.Started = task.Downloader.Started
		if task.Downloader.Finished || task.Downloader.Error != nil {
			ti.Finished = true
		}
		if task.Downloader.Error != nil {
			ti.Error = task.Downloader.Error.Error()
		}
		ti.CurrentSegment.Num = task.Downloader.CurrentSegment.Num
		ti.CurrentSegment.Size = task.Downloader.CurrentSegment.Size
		ti.CurrentSegment.Url = task.Downloader.CurrentSegment.Url
		ti.CurrentSegment.GotBytes = task.Downloader.CurrentSegment.GotBytes
		ti.DownloadedDuration = task.Downloader.DownloadedDuration
		ti.GotBytes = task.Downloader.GotBytes
		if task.Downloader.Playlist != nil {
			ti.SegmentsCount = task.Downloader.Playlist.SegmentsCount
			ti.SegmentsDuration = task.Downloader.Playlist.SegmentsDuration
		}
	}
	return ti
}

func (task *Task) IsError() bool {
	if task.Downloader != nil && task.Downloader.Error != nil {
		return true
	}
	return false
}
