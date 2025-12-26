package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	address     = flag.String("address", "0.0.0.0", "Listening address")
	port        = flag.String("port", "8080", "Listening port")
	sslPort     = flag.String("sslPort", "10433", "SSL listening port")
	path        = flag.String("path", "/", "URL path")
	deny        = flag.String("deny", "", "Sensitive directory or file patterns to be denied when serving directory (comma separated)")
	status      = flag.Int("status", 200, "Returned HTTP status code")
	cert        = flag.String("cert", "cert.pem", "SSL certificate path")
	key         = flag.String("key", "key.pem", "SSL private Key path")
	proxy       = flag.String("proxy", "", "URL prefixes to be proxied to another server e.g. /api=>http://localhost:3000 will forward all requests starting with /api to http://localhost:3000 (comma separated)")
	corsOrigin  = flag.String("corsOrigin", "", "Allow CORS requests from this origin (can be '*')")
	corsMethods = flag.String("corsMethods", "POST, GET, OPTIONS, PUT, DELETE", "Allowed CORS methods")
	corsHeaders = flag.String("corsHeaders", "Content-Type, Authorization, X-Requested-With", "Allowed CORS headers")
	contentType = flag.String("contentType", "", "Set response Content-Type")
	versionFlag = flag.Bool("version", false, "prints current version")
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "goreleaser"
)

type bytesHandler []byte

func (h bytesHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(*status)
	w.Write(h)
}

type proxyHandler struct {
	prefix   string
	proxyURL string
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *contentType != "" {
			w.Header().Set("Content-Type", *contentType)
		}
		if corsOrigin != nil && *corsOrigin != "" {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", *corsOrigin)
			w.Header().Set("Access-Control-Allow-Methods", *corsMethods)
			w.Header().Set("Access-Control-Allow-Headers", *corsHeaders)

			// Check if the request is a preflight request and handle it.
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

type echoHandler struct{}

func (eh *echoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var buffer bytes.Buffer
	multiWriter := io.MultiWriter(w, &buffer)

	// Buffer body
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body.Close()
	}

	// Write REQUEST section
	fmt.Fprintf(multiWriter, "=== REQUEST ===\n")
	fmt.Fprintf(multiWriter, "Method: %s\n", r.Method)
	fmt.Fprintf(multiWriter, "Path: %s\n", r.URL.Path)
	if r.URL.RawQuery != "" {
		fmt.Fprintf(multiWriter, "Query: %s\n", r.URL.RawQuery)
	}
	fmt.Fprintf(multiWriter, "Host: %s\n", r.Host)
	fmt.Fprintf(multiWriter, "Protocol: %s\n", r.Proto)
	fmt.Fprintf(multiWriter, "\n")

	// Write HEADERS section
	fmt.Fprintf(multiWriter, "=== HEADERS ===\n")
	for name, values := range r.Header {
		for _, value := range values {
			fmt.Fprintf(multiWriter, "%s: %s\n", name, value)
		}
	}
	fmt.Fprintf(multiWriter, "\n")

	// Write BODY section
	fmt.Fprintf(multiWriter, "=== BODY ===\n")
	if len(bodyBytes) > 0 {
		multiWriter.Write(bodyBytes)
		fmt.Fprintf(multiWriter, "\n")
	} else {
		fmt.Fprintf(multiWriter, "(empty)\n")
	}

	// Log everything that was written
	log.Print(buffer.String())
}

func (ph *proxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Construct target URL with proper prefix removal and query parameter preservation
	targetPath := strings.TrimPrefix(req.URL.Path, ph.prefix)
	targetURL := ph.proxyURL + targetPath
	if req.URL.RawQuery != "" {
		targetURL += "?" + req.URL.RawQuery
	}
	if req.URL.Fragment != "" {
		targetURL += "#" + req.URL.Fragment
	}

	request, err := http.NewRequest(req.Method, targetURL, req.Body)
	if err != nil {
		log.Println("error creating proxy request: ", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	request.Header = req.Header

	response, err := client.Do(request)
	if err != nil {
		log.Print("error response from proxy: ", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	// Copy all headers from proxied response
	for key, values := range response.Header {
		w.Header().Del(key)
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Override with our CORS settings if configured
	if *corsOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", *corsOrigin)
		w.Header().Set("Access-Control-Allow-Methods", *corsMethods)
		w.Header().Set("Access-Control-Allow-Headers", *corsHeaders)
	}

	// Write status code from proxied response
	w.WriteHeader(response.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, response.Body); err != nil {
		log.Print("error copying the response from proxy: ", err)
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

func parseProxy(flagStr string) (handlers []*proxyHandler) {
	proxyList := strings.Split(flagStr, ",")
	for _, proxyRedirect := range proxyList {
		proxyElements := strings.Split(proxyRedirect, "=>")
		if len(proxyElements) == 2 {
			prefix := strings.TrimSpace(proxyElements[0])
			proxyURL := strings.TrimSpace(proxyElements[1])
			if strings.HasPrefix(prefix, "/") && strings.HasPrefix(proxyURL, "http") {
				handlers = append(handlers, &proxyHandler{
					prefix:   prefix,
					proxyURL: proxyURL,
				})
			} else {
				log.Printf("bad proxy pair: %s=>%s", prefix, proxyURL)
			}
		}
	}
	return
}

type protectedFileSystem struct {
	fs http.FileSystem
}

func (pfs protectedFileSystem) Open(path string) (http.File, error) {
	if isDenied(path, *deny) {
		return nil, os.ErrPermission
	}
	return pfs.fs.Open(path)
}

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("spark version %s, commit %s, built at %s by %s\n", version, commit, date, builtBy)
		os.Exit(0)
	}

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
			handler = http.StripPrefix(*path, http.FileServer(protectedFileSystem{http.Dir(body)}))
		case mode.IsRegular():
			if content, err := os.ReadFile(body); err != nil {
				log.Fatal("Error reading file: ", err)
			} else {
				handler = bytesHandler(content)
			}
		}
	} else {
		handler = bytesHandler(body)
	}
	http.Handle(*path, middleware(handler))

	http.Handle("/echo", middleware(&echoHandler{}))

	if proxy != nil {
		proxyHandlers := parseProxy(*proxy)
		for _, ph := range proxyHandlers {
			log.Printf("sending %s to %s", ph.prefix, ph.proxyURL)
			http.Handle(ph.prefix, middleware(ph))
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
