# spark

Emergency web server

For those occasions when your webserver is down and you want to display a quick maintainance note. Or just want to quickly demo a static site. Or whatever :)

It can take a directory, a file or directly the body string.


```
❯ spark -h
Usage of spark:
  -address="0.0.0.0": Listening address
  -port="8080": Listening port
  -cert="cert.pem": SSL certificate path
  -key="key.pem": SSL private Key path
  -sslPort="10433": SSL listening port
  -status=200: Returned HTTP status code

```

## examples

```
$ spark message.html
$ spark "<h1>Out of order</h1><p>Working on it...</p>"
$ spark static_site/
$ spark -port 80 -sslPort 443 "<h1>Ooops!</h1>"
```

To quickly generate a ssl certificate run:

```
go run $GOROOT/src/crypto/tls/generate_cert.go --host="localhost"
```

## install
- from source
```
go get github.com/rif/spark
```
- static binaries (linux/arm/osx/windows):

<a href="https://copy.com/ASm3M2aWp0aU9kN4/spark" target="_blank">Binary downloads</a>

## crossbuild

```
docker build --rm -t crossbuild crossbuild/
# creates a ubuntu docker container with golang crossbuild

docker run --rm -itv $GOPATH:/go crossbuild /go/src/github.com/rif/spark/crossbuild.sh
# creates a ./build directory that will be shared with the world
```
