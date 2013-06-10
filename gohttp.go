package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const htmlTpl = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{.CurrentURI}} - gohttp</title></head>
<body>
<a href="{{.ParentURI}}"> {{.ParentURI}} </a> | <a href="{{.CurrentURI}}">{{.CurrentURI}}</a>
<ul>
   {{range .files}}
      <li><a href="{{.URI}}">{{.Name}}
      {{if .Size }}
      <small>({{.Size}})</small>
      {{end}}
      </a></li>
   {{end}}</ul>
</body></html>`

var (
	port    = 8080
	webroot = "."
	tmp     = template.Must(template.New("index").Parse(htmlTpl))
)

////////////////////////////////////////////////////////////////////////////////////////////
func init() {
	wd, _ := os.Getwd()
	flag.IntVar(&port, "port", port, "The port (default is 8080)")
	flag.StringVar(&webroot, "webroot", wd, "Web root directory (default is current work directory)")
}

//////////////////////////////////////////////////////////////////////////////////////////
type Server struct {
	port    int
	webroot string
}

type Item struct {
	Name  string
	Title string
	URI   string
	Size  int64
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func getHostIPs() string {
	switch runtime.GOOS {
	case "linux":
		{
			ifi, err := net.InterfaceByName("wlan0")
			if err != nil {
				return ""
			}
			addrs, err := ifi.Addrs()
			if err != nil {
				return ""
			}
			for _, addr := range addrs {
				ip := addr.String()
				if !strings.Contains(ip, ":") && !strings.HasPrefix(ip, "127") {
					end := len(ip) - 3
					return ip[0:end]
				}
			}
			return ""
		}
	default:
		{
			name, err := os.Hostname()
			if err != nil {
				return ""
			}

			addrs, err := net.LookupHost(name)
			if err != nil {
				return ""
			}
			ips, err := net.LookupIP(name)
			fmt.Printf("ips:%v\n", ips)
			for _, addr := range addrs {
				if !strings.Contains(addr, ":") && !strings.HasPrefix(addr, "127") {
					return addr
				}
			}
			return strings.Join(addrs, ", ")
		}
	}
}

func (server *Server) Start() {
	server.router()

	fmt.Printf("Serving HTTP on %s port %d from \"%s\" ... \n",
		getHostIPs(), server.port, server.webroot,
	)

	addr := fmt.Sprintf(":%v", server.port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		}
	}()

	server.handler(w, req)
}

func (server *Server) router() {
	http.Handle("/", server)
}

func (server *Server) handler(w http.ResponseWriter, req *http.Request) {
	uri := req.RequestURI      // 请求的URI, 如http://localhost:8080/hello -> /hello
	if uri == "/favicon.ico" { // 不处理
		return
	}

	log.Printf(`%s "%s" from %v`, req.Method, req.RequestURI, req.RemoteAddr)

	fullpath, relpath := server.requestURIToFilepath(uri)
	log.Printf("\tTo Filepath:%v", fullpath)

	file, err := os.Open(fullpath)
	if err != nil || os.IsNotExist(err) { // 文件不存在
		log.Println("\tNotFound")
		http.NotFound(w, req)
	} else {
		stat, _ := file.Stat()
		if stat.IsDir() {
			log.Printf("\tProcess Dir...")
			server.processDir(w, file, fullpath, relpath)
		} else {
			log.Printf("\tSend File...")
			server.sendFile(w, file, fullpath, relpath)
		}
	}

	log.Printf("END")
}

func (server *Server) requestURIToFilepath(uri string) (fullpath string, relpath string) {
	unescapeIt, _ := url.QueryUnescape(uri)

	relpath = unescapeIt
	log.Printf("\tUnescape URI:%v", relpath)

	fullpath = filepath.Join(server.webroot, relpath[1:])

	return
}

func (server *Server) processDir(w http.ResponseWriter, dir *os.File, fullpath string, relpath string) {
	w.Header().Set("Content-type", "text/html; charset=UTF-8")
	fis, err := dir.Readdir(-1)
	checkError(err)

	items := make([]Item, 0, len(fis))
	for _, fi := range fis {
		item := Item{
			Name:  fi.Name(),
			Title: fi.Name(),
			URI:   path.Join(relpath, fi.Name()),
			Size:  fi.Size(),
		}
		items = append(items, item)
	}

	tmp.Execute(w, map[string]interface{}{
		"ParentURI":  path.Dir(relpath),
		"CurrentURI": relpath,
		"files":      items,
	})
}

func (server *Server) sendFile(w http.ResponseWriter, file *os.File, fullpath string, relpath string) {
	if mimetype := mime.TypeByExtension(path.Ext(file.Name())); mimetype != "" {
		w.Header().Set("Content-Type", mimetype)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	statinfo, _ := file.Stat()
	w.Header().Set("Content-Length", fmt.Sprintf("%v", statinfo.Size()))
	io.Copy(w, file)
}

////////////////////////////////////////////////////////////////////////////////////////////////

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU()*2 - 1)
	flag.Parse()
	server := &Server{port: port, webroot: webroot}
	server.Start()
}
