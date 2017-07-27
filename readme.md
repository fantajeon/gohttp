gohttp: go simple http server
---------------------

Golang implementation replace "python -m SimpleHTTPServer"

### Usage

```
When url = "http://localhost:8080/a.html?param1=val1&param2=val2", request file is /a.html.

$ go get -u github.com/fantajeon/gohttp
$ go install github.com/fantajeon/gohttp/gohttp

$ gohttp --help

$ gohttp
Serving HTTP on 192.168.1.103 port 8080 from "/home/fantajeon/workspace/work" ...

$ gohttp -d=/home -p=9000
Serving HTTP on 192.168.1.128 port 9000 from "/home" ...
```

### License

Distributed under the [Apache License Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).
