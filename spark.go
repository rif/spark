package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	address = flag.String("address", "0.0.0.0", "Listening address")
	port    = flag.String("port", "8080", "Listening port")
)

type bytesHandler []byte

func (h bytesHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Write(h)
}

func main() {
	flag.Parse()
	listen := *address + ":" + *port
	body := flag.Arg(0)
	if body == "" {
		body = "<h1>Spark!</h1>"
	}
	var handler http.Handler
	if fi, err := os.Stat(body); err == nil {
		switch mode := fi.Mode(); {
		case mode.IsDir():
			handler = http.FileServer(http.Dir(body))
		case mode.IsRegular():
			if content, err := ioutil.ReadFile(body); err != nil {
				log.Fatal("Error reading file: ", err)
			} else {
				handler = bytesHandler(content)
			}
		}
	} else {
		handler = bytesHandler(body)
	}
	log.Fatal(http.ListenAndServe(listen, handler))
}
