package main

import (
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

const VERSION = `0.2.4`

var (
	ErrorLog = log.New(os.Stderr, `error#`, log.Lshortfile)
	DebugLog = log.New(os.Stdout, `debug#`, log.Lshortfile)

	//go:embed index.html
	indexTemplate string

	//go:embed task.html
	taskTemplate string
)

func getTemplate(fileName, stringTemplate string) *template.Template {
	t, err := template.ParseFiles(fileName)
	if err != nil {
		t, err = template.New(``).Parse(stringTemplate)
		if err != nil {
			ErrorLog.Fatalln(err.Error())
		}
	}
	return t
}

func helpText() {
	fmt.Println(`bla-bla-bla`)
	flag.PrintDefaults()
}

func main() {
	help := flag.Bool("h", false, "print this help")
	listen := flag.String("l", ":80", "listen address")
	ver := flag.Bool("v", false, "Show version")
	flag.Parse()

	if *help {
		helpText()
		os.Exit(0)
	}

	if *ver {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	core := NewCore()

	server := http.Server{Addr: *listen}
	http.HandleFunc("/add", core.addHandler)
	http.HandleFunc("/{$}", core.indexHandler)
	http.HandleFunc("/{task}/{$}", core.taskHandler)
	http.HandleFunc(`/favicon.ico`, http.NotFound)
	if err := server.ListenAndServe(); err != nil {
		ErrorLog.Fatalln(err.Error())
	}
}
