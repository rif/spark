package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
    address = flag.String("address", "0.0.0.0", "Listening address")
    port    = flag.String("port", "8080", "Listening port")
    sslPort = flag.String("sslPort", "10433", "SSL listening port")
    path    = flag.String("path", "/", "URL path")
    deny    = flag.String("deny", "", "Sensitive directory or file patterns to be denied when serving directory (comma separated)")
    status  = flag.Int("status", 200, "Returned HTTP status code")
    cert    = flag.String("cert", "cert.pem", "SSL certificate path")
    key     = flag.String("key", "key.pem", "SSL private Key path")
    proxy = flag.String("proxy", "", "URL prefixes to be proxied to another server e.g. /api=>http://localhost:3000 will forward all requests starting with /api to http://localhost:3000 (comma separated)")
)

type bytesHandler []byte

func (h bytesHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(*status)
    w.Write(h)
}

type proxyHandler string

func (p proxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    client := &http.Client{
		Timeout: 5*time.Second,
	}
	request, err := http.NewRequest(req.Method, string(p), req.Body)
	if err != nil {
		log.Println(err)
	}
	request.Header = req.Header

    response, err := client.Do(request)
    if err != nil {
        log.Print("error response form proxy: ", err)
    } else {
		if _, err:= io.Copy(w,response.Body); err != nil {
			log.Print("error copyting the response from proxy: ", err)
		}
    }
}

func isDenied(path, denyList string) bool {
    if len(denyList) == 0 {
        return false
    }
    for _, pathElement := range strings.Split(path, string(filepath.Separator)) {
        for _, denyElement := range strings.Split(denyList, ",") {
            match, err := filepath.Match(strings.TrimSpace(denyElement), pathElement)
            if err != nil {
                log.Print("error matching file path element: ", err)
            }
            if match {
                return true
            }
        }
    }
    return false
}

func parseProxy(flagStr string)map[string]string{
    proxyMap:= make(map[string]string)
    proxyList := strings.Split(flagStr, ",")
    for _, proxyRedirect := range proxyList {
        proxyElements := strings.Split(proxyRedirect, "=>")
        if len(proxyElements) == 2{
            prefix := strings.TrimSpace(proxyElements[0])
            proxyURL:= strings.TrimSpace(proxyElements[1])
            if strings.HasPrefix(prefix,"/") && strings.HasPrefix(proxyURL,"http") {
                proxyMap[prefix] = proxyURL
            } else {
                log.Printf("bad proxy pair: %s=>%s", prefix,proxyURL)
            }
        }
    }
    return proxyMap
}

type protectdFileSystem struct {
    fs http.FileSystem
}

func (pfs protectdFileSystem) Open(path string) (http.File, error) {
    if isDenied(path, *deny) {
        return nil, os.ErrPermission
    }
    return pfs.fs.Open(path)
}

func main() {
    flag.Parse()
    listen := *address + ":" + *port
    listenTLS := *address + ":" + *sslPort
    body := flag.Arg(0)
    if body == "" {
        body = "."
    }
    var handler http.Handler
    if fi, err := os.Stat(body); err == nil {
        switch mode := fi.Mode(); {
        case mode.IsDir():
            if *deny == "" {
                log.Print("Warning: serving files without any filter!")
            }
            handler = http.StripPrefix(*path, http.FileServer(protectdFileSystem{http.Dir(body)}))
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
    http.Handle(*path, handler)

    if (proxy != nil){
		proxyMap := parseProxy(*proxy)
        for prefix, proxyURL := range proxyMap {
            log.Printf("sending %s to %s", prefix,proxyURL)
            http.Handle(prefix, proxyHandler(proxyURL))
        }
    }

    go func() {
        if _, err := os.Stat(*cert); err != nil {
            return
        }
        if _, err := os.Stat(*key); err != nil {
            return
        }
        log.Fatal(http.ListenAndServeTLS(listenTLS, *cert, *key, nil))
    }()
    log.Printf("Serving %s on %s%s...", body, listen, *path)
    log.Fatal(http.ListenAndServe(listen, nil))
}
