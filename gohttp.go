package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

var (
	port    = 8080
	webroot = "."
)

var tmp = template.Must(template.New("index").Parse(html))

const html = `
<a href="{{.ParentURI}}"> {{.ParentURI}} </a> | <a href="{{.CurrentURI}}">{{.CurrentURI}}</a>
<ul>
   {{range .files}}
      <li><a href="{{.URI}}">{{.Name}}
      {{if .Size }}
      <small>({{.Size}})</small>
      {{end}}
      </a></li>
   {{end}}
</ul>`

////////////////////////////////////////////////////////////////////////////////////////////
func init() {
	wd, _ := os.Getwd()
	log.Printf("Current dir:%v", wd)
	log.Printf("PathSeparator:%c", os.PathSeparator)

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

func (server *Server) Start() {
	log.Printf("Port:%v", server.port)
	log.Printf("Webroot:%v", server.webroot)

	server.router()

	addr := fmt.Sprintf(":%v", server.port)
	fmt.Printf("Start server at :%v\n", server.port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Errorf("%v", err)
	}
}

func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("<< Request from %v", req.RemoteAddr)

	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		}
	}()

	server.handler(w, req)

	log.Printf("End Request>>")
}

func (server *Server) router() {
	http.Handle("/", server)
}

func (server *Server) handler(w http.ResponseWriter, req *http.Request) {
	uri := req.RequestURI      // 请求的URI, 如http://localhost:8080/hello -> /hello
	if uri == "/favicon.ico" { // 不处理
		return
	}

	fullpath, relpath := server.requestURIToFilepath(uri)
	log.Printf("\tTo Filepath:%v", fullpath)

	file, err := os.Open(fullpath)
	if err != nil || os.IsNotExist(err) { // 文件不存在
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
}

func (server *Server) requestURIToFilepath(uri string) (fullpath string, relpath string) {
	unescapeIt, _ := url.QueryUnescape(uri)

	relpath = unescapeIt
	log.Printf("\tUnescape URI:%v", relpath)

	fullpath = filepath.Join(server.webroot, relpath[1:])
	//log.Printf("base path:%v, dir path:%v, ext path:%v\n", path.Base(fullpath), path.Dir(fullpath), path.Ext(fullpath))

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
