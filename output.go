package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Output struct {
	Path string
	Created bool
	Opened bool
	fd io.WriteCloser
}

func (output *Output) Write(p []byte) (n int, err error) {
	if output.fd == nil {
		return 0, errors.New(io.ErrClosedPipe.Error())
	}
	return output.fd.Write(p)
}

func (output *Output) Close() error {
	if output.fd == nil {
		return errors.New(io.ErrClosedPipe.Error())
	}
	if output.Path == `-` {
		output.fd = nil
		return nil
	}

	closeErr := output.fd.Close()
	if closeErr != nil { log.Println(closeErr.Error()) }

	output.fd = nil

	if !output.Created { return closeErr }
	output.Created = false

	fileInfo, err := os.Stat(output.Path)
	if err != nil {
		log.Println(err.Error())
		return closeErr
	}
	if fileInfo.Size() == 0 {
		if err := os.Remove(output.Path); err!=nil { log.Println(err.Error()) }
	}

	return closeErr
}

func (output *Output) SetPath(path string, overwrite string) {
	if output.Created {
		log.Printf("Can't set path `%s` because `%s` already created!\n", path, output.Path)
		return
	}
	if path == `` { path = fmt.Sprintf("%d.mp4", time.Now().Unix()) }
	if overwrite == `uniq` { path = output.getUniqName(path) }
	output.Path = path
}

func (output *Output) getUniqName(path string) string {
	if path == `-` { return path }
	for output.isExists(path) {
		dir := filepath.Dir(path)
		base := filepath.Base(path)
		base = `_` + base
		path = filepath.Join(dir, base)
	}
	return path
}

func (output *Output) isExists(path string) bool {
	if path == `-` { return false }
	_, err := os.Stat(path)
	if err == nil { return true }
	return os.IsExist(err)
}

func (output *Output) Open(overwrite string) error {
	if output.Opened {
		msg := fmt.Sprintf("'%s' already opened")
		log.Println(msg)
		return errors.New(msg)
	}

	if output.Path == `-` {
		output.fd = os.Stdout
		output.Opened = true
		return nil
	}

	flags := os.O_CREATE|os.O_WRONLY
	if overwrite == `overwrite` { flags = os.O_CREATE|os.O_WRONLY|os.O_TRUNC }
	f, err := os.OpenFile(output.Path, flags, 0755)
	if err != nil {
		log.Printf("Can't open '%s': %s\n", output.Path, err.Error())
		return err
	}
	//TODO: set lock to file

	output.fd = f
	output.Created = true
	output.Opened = true
	return nil
}


func NewOutput(args *Args) (*Output, error) {
	output := new(Output)
	output.SetPath(*args.Output, *args.Overwrite)
	if err := output.Open(*args.Overwrite); err != nil {
		return nil, err
	}
	return output, nil
}