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
	mock        = flag.String("mock", "", "Directory containing mock responses")
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

type mockHandler struct {
	mockDir      string
	endpointPath string
}

func (mh *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Construct the directory path for this endpoint
	dirPath := filepath.Join(mh.mockDir, mh.endpointPath)

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error reading mock directory %s: %v", dirPath, err)
		return
	}

	// Find matching method file (case-insensitive)
	var matchedFile string
	var statusCode int
	allowedMethods := []string{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		// Parse METHOD or METHOD_STATUS format
		parts := strings.Split(fileName, "_")
		method := strings.ToUpper(parts[0])
		allowedMethods = append(allowedMethods, method)

		if strings.ToUpper(r.Method) == method {
			matchedFile = fileName
			// Parse status code if present
			if len(parts) > 1 {
				if code, err := filepath.Match("[0-9]*", parts[1]); err == nil && code {
					fmt.Sscanf(parts[1], "%d", &statusCode)
				}
			}
			if statusCode == 0 {
				statusCode = 200 // Default status code
			}
			break
		}
	}

	// If no matching method found, return 405
	if matchedFile == "" {
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read file content
	filePath := filepath.Join(dirPath, matchedFile)
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error reading mock file %s: %v", filePath, err)
		return
	}

	// Detect content type from file extension if present
	ext := filepath.Ext(matchedFile)
	if ext == "" {
		// Check for common extensions after the method name
		nameParts := strings.Split(matchedFile, ".")
		if len(nameParts) > 1 {
			ext = "." + nameParts[len(nameParts)-1]
		}
	}

	// Set content type based on extension
	switch ext {
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".xml":
		w.Header().Set("Content-Type", "application/xml")
	case ".html":
		w.Header().Set("Content-Type", "text/html")
	case ".txt":
		w.Header().Set("Content-Type", "text/plain")
	default:
		// Try to detect from content
		contentType := http.DetectContentType(content)
		w.Header().Set("Content-Type", contentType)
	}

	// Write response
	w.WriteHeader(statusCode)
	w.Write(content)
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

func walkMockDir(mockDir string) (handlers []*mockHandler) {
	// Check if mock directory exists
	if _, err := os.Stat(mockDir); os.IsNotExist(err) {
		log.Printf("Warning: mock directory does not exist: %s", mockDir)
		return
	}

	// Walk the directory tree
	err := filepath.Walk(mockDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == mockDir {
			return nil
		}

		// Only process directories
		if !info.IsDir() {
			return nil
		}

		// Check if this directory contains any HTTP verb files
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil // Skip directories we can't read
		}

		hasVerbFiles := false
		for _, entry := range entries {
			if !entry.IsDir() {
				// Check if filename starts with a common HTTP verb
				name := strings.ToUpper(entry.Name())
				if strings.HasPrefix(name, "GET") ||
					strings.HasPrefix(name, "POST") ||
					strings.HasPrefix(name, "PUT") ||
					strings.HasPrefix(name, "DELETE") ||
					strings.HasPrefix(name, "PATCH") ||
					strings.HasPrefix(name, "HEAD") ||
					strings.HasPrefix(name, "OPTIONS") {
					hasVerbFiles = true
					break
				}
			}
		}

		if hasVerbFiles {
			// Calculate the endpoint path relative to mock directory
			relPath, err := filepath.Rel(mockDir, path)
			if err != nil {
				return nil
			}

			// Convert to URL path (use forward slashes)
			endpointPath := "/" + filepath.ToSlash(relPath)

			handlers = append(handlers, &mockHandler{
				mockDir:      mockDir,
				endpointPath: relPath,
			})

			log.Printf("Registered mock endpoint: %s", endpointPath)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error walking mock directory: %v", err)
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

	// Register mock handlers first (if mock mode is enabled) to ensure priority
	if mock != nil && *mock != "" {
		mockHandlers := walkMockDir(*mock)
		for _, mh := range mockHandlers {
			// Convert endpoint path to URL path with forward slashes
			urlPath := "/" + filepath.ToSlash(mh.endpointPath)
			http.Handle(urlPath, middleware(mh))
		}
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
