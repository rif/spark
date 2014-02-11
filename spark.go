package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	address = flag.String("address", "0.0.0.0", "Listening address")
	port    = flag.String("port", "8080", "Listening port")
	dir     = flag.String("dir", "", "Serve files from directory")
	file    = flag.String("file", "", "Serve the file content")
	body    = flag.String("body", "", "Serve the given body")
)

type StringHandler struct {
	body string
}

func (sh StringHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, sh.body)
}

func HelloServer(w http.ResponseWriter, req *http.Request) {

}

func main() {
	flag.Parse()
	listen := *address + ":" + *port
	var handler http.Handler
	if *dir != "" {
		handler = http.FileServer(http.Dir(*dir))
	}
	if *file != "" {
		content, err := ioutil.ReadFile(*file)
		if err != nil {
			log.Fatal("Error reading file: ", err)
		}
		log.Print("file: ", string(content))
		handler = StringHandler{body: string(content)}
	}
	if *body != "" {
		log.Print("body: ", *body)
		handler = StringHandler{body: *body}
	}
	log.Fatal(http.ListenAndServe(listen, handler))
}
