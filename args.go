package main

import (
	"fmt"
	"path"
	"time"
)

type Args struct {
	Output *string
	outputPath string
}

func (args *Args) GetOutputPath() string {
	if args.outputPath != `` { return args.outputPath }
	outputPath := *args.Output
	if outputPath == `-` { return outputPath }
	if outputPath == `` {
		outputPath = fmt.Sprintf("%d.mp4", time.Now().Unix())
	}
	outputPath = path.Clean(outputPath)
	args.outputPath = outputPath
	return outputPath
}
