package main

import (
	"io"
	"net/http"
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/parnurzeal/gorequest"
	"time"
	"fmt"
	"strings"
	"strconv"
	"os"
	"net"
)

var (
	ngxStatusUrl = kingpin.Arg("url", "Nginx status url.").Required().String()
	listenAddr = kingpin.Arg("unix-sock", "Exporter listen addr. Default is /dev/shm/nginx_exporter.sock").
		String()
)

func Substr(str string, start, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}

	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}

	return string(rs[start:end])
}

func metrics(w http.ResponseWriter, req *http.Request) {
	ret := ""
	r := gorequest.New()
	_, body, errs := r.Retry(3, time.Second * 3,
		http.StatusBadRequest, http.StatusInternalServerError).Get(url).End()
	if errs == nil {
		s := body
		s = strings.TrimRight(s, "\n")
		l := strings.Split(s, "\n")
		if len(l) != 4 {
			io.WriteString(w, ret)
			return
		}

		active := strings.TrimLeft(l[0], "Active connections:")
		active = strings.TrimSpace(active)

		reqsStr := strings.TrimLeft(l[2], " ")
		reqsStr = strings.TrimRight(reqsStr, " ")
		reqs := strings.Split(reqsStr, " ")
		accepted := strings.TrimSpace(reqs[0])
		handled := strings.TrimSpace(reqs[1])
		requests := strings.TrimSpace(reqs[2])

		connsStr := strings.TrimLeft(l[3], "Reading:")
		connsStr = strings.TrimLeft(connsStr, " ")
		idx1 := strings.Index(connsStr, " W")
		reading := Substr(connsStr, 0, idx1)
		left := Substr(connsStr, strings.Index(connsStr, ":") + 1, 128)
		lefts := strings.Split(left, "Waiting:")
		writing := strings.TrimSpace(lefts[0])
		waiting := strings.TrimSpace(lefts[1])

		activeVal, _ := strconv.ParseFloat(active, 64)
		acceptedVal, _ := strconv.ParseFloat(accepted, 64)
		handledVal, _ := strconv.ParseFloat(handled, 64)
		requestsVal, _ := strconv.ParseFloat(requests, 64)
		readingVal, _ := strconv.ParseFloat(reading, 64)
		writingVal, _ := strconv.ParseFloat(writing, 64)
		waitingVal, _ := strconv.ParseFloat(waiting, 64)
		ret = fmt.Sprintf("nginx_server_connections{status=\"active\"} %g\n", activeVal)
		ret += fmt.Sprintf("nginx_server_connections{status=\"accepted\"} %g\n", acceptedVal)
		ret += fmt.Sprintf("nginx_server_connections{status=\"handled\"} %g\n", handledVal)
		ret += fmt.Sprintf("nginx_server_connections{status=\"requests\"} %g\n", requestsVal)
		ret += fmt.Sprintf("nginx_server_connections{status=\"reading\"} %g\n", readingVal)
		ret += fmt.Sprintf("nginx_server_connections{status=\"writing\"} %g\n", writingVal)
		ret += fmt.Sprintf("nginx_server_connections{status=\"waiting\"} %g\n", waitingVal)
	}
	io.WriteString(w, ret)
}

var url string

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()
	addr := ""
	if ngxStatusUrl == nil {
		panic("No nginx status url")
		return
	}
	url = *ngxStatusUrl

	if listenAddr != nil {
		addr = *listenAddr
	} else {
		addr = "/dev/shm/nginx_exporter.sock"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("", metrics)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Memcached Exporter</title></head>
             <body>
             <h1>Memcached Exporter</h1>
             <p><a href='` + "/metrics" + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	server := http.Server{
		Handler: mux, // http.DefaultServeMux,
	}
	os.Remove(addr)

	listener, err := net.Listen("unix", addr)
	if err != nil {
		panic(err)
	}
	server.Serve(listener)
}