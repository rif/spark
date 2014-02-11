# spark

Emergency web server


```
❯ spark -h
Usage of spark:
  -address="0.0.0.0": Listening address
  -port="8080": Listening port
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

[linux64](https://raw.github.com/rif/spark/master/spark_linux64.xz)
[osx](https://raw.github.com/rif/spark/master/spark_osx.xz)

