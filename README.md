# spark

Emergency web server

For those occasions when your webserver is down and you want to display a quick maintainance note. Or just want to quickly demo a static site. Or whatever :)

It can take a directory, a file or directly the body string.


```
❯ spark -h
Usage of spark:
  -address="0.0.0.0": Listening address
  -port="8080": Listening port
  -status=200: Returned HTTP status code
```

## examples

```
$ spark message.html
$ spark "<h1>Out of order</h1><p>Working on it...</p>"
$ spark static_site/
# spark -port 80 "<h1>Ooops!</h1>"
```

## install
- from source
```
go get github.com/rif/spark
```
- static binaries:

[linux64](https://github.com/rif/spark/releases/download/v1.1/linux64.binary)
[osx](https://github.com/rif/spark/releases/download/v1.1/OSX.binary)

```
tar xvf spark_linux64.xz
mv spark_linux64 somewhere_in_your_path/spark
spark away!
```
