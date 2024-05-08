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
    	Allowd CORS methods (default "POST, GET, OPTIONS, PUT, DELETE")
  -corsOrigin string
    	Allow CORS request from this origin (can be '*')
  -deny string
    	Sensitive directory or file patterns to be denied when serving directory (comma separated)
  -key string
    	SSL private Key path (default "key.pem")
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

To quickly generate a ssl certificate run:

```
go run $GOROOT/src/crypto/tls/generate_cert.go --host="localhost"
```
