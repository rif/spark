# spark

Emergency web server

For those occasions when your webserver is down and you want to display a quick maintainance note. Or just want to quickly demo a static site. Or whatever :)

It can take a directory, a file or directly the body string. The `-proxy` flag can be useful when used as a development server.


```
❯ spark -h
Usage of spark:
  -address string
    	Listening address (default "0.0.0.0")
  -cert string
    	SSL certificate path (default "cert.pem")
  -contentType string
    	Set response Content-Type
  -corsHeaders string
    	Allowed CORS headers (default "Content-Type, Authorization, X-Requested-With")
  -corsMethods string
    	Allowed CORS methods (default "POST, GET, OPTIONS, PUT, DELETE")
  -corsOrigin string
    	Allow CORS requests from this origin (can be '*')
  -deny string
    	Sensitive directory or file patterns to be denied when serving directory (comma separated)
  -key string
    	SSL private Key path (default "key.pem")
  -mock string
    	Directory containing mock responses
  -path string
    	URL path (default "/")
  -port string
    	Listening port (default "8080")
  -proxy string
    	URL prefixes to be proxied to another server e.g. /api=>http://localhost:3000 will forward all requests starting with /api to http://localhost:3000 (comma separated)
  -sslPort string
    	SSL listening port (default "10433")
  -status int
    	Returned HTTP status code (default 200)
  -version
    	prints current version
```

## install
- from source
```
go get github.com/rif/spark
```
- static binaries (linux/arm/osx/windows):
  <a href="https://github.com/rif/spark/releases" target="_blank">Binary downloads</a>

## examples

```
$ spark message.html
$ spark "<h1>Out of order</h1><p>Working on it...</p>"
$ spark static_site/
$ spark -port 80 -sslPort 443 "<h1>Ooops!</h1>"
$ spark -deny ".git*,LICENSE" ~/go/rif/spark
$ spark -proxy "/api=>http://localhost:9090/api" .
$ spark -port 9000 -corsOrigin "https://www.mydomain.com" -contentType "application/json" '{"message":"Hello"}'
```

## new features

### echo endpoint
The `/echo` endpoint returns request details (method, headers, body) for debugging:
```
$ spark "Hello" &
$ curl -X POST http://localhost:8080/echo -d '{"test": "data"}'
```

### mock server
Create mock API responses using directory structure:
```
# Directory structure defines endpoints
mock/
  users/
    GET              # responds to GET /users
    POST_201         # responds to POST /users with 201 status
  api/
    products/
      GET            # responds to GET /api/products

# Start mock server
$ spark -mock mock/

# Test endpoints
$ curl http://localhost:8080/users
$ curl -X POST http://localhost:8080/users
```

Files are named after HTTP verbs (case-insensitive). Add `_STATUS` suffix for custom status codes (e.g., `POST_201`, `DELETE_204`). Returns 405 Method Not Allowed for unsupported methods.

## ssl certificate

To quickly generate a ssl certificate run:

```
go run $GOROOT/src/crypto/tls/generate_cert.go --host="localhost"
```
