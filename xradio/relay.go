package main

import (
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/koykov/conply"
)

type Relay interface {
	io.Closer
	Wrap(target string) string
}

type dummyRelay struct{}

func (dummyRelay) Wrap(target string) string { return target }
func (dummyRelay) Close() error              { return nil }

type relay struct {
	srv    *http.Server
	client *http.Client
	addr   string
}

func NewRelay(client *http.Client) (Relay, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	r := &relay{client: client, addr: ln.Addr().String()}
	mux := http.NewServeMux()
	mux.HandleFunc("/", r.handle)
	r.srv = &http.Server{Handler: mux}
	go func() { _ = r.srv.Serve(ln) }()
	return r, nil
}

func (r *relay) Wrap(target string) string {
	return "http://" + r.addr + "/?u=" + url.QueryEscape(target)
}

func (r *relay) Close() error { return r.srv.Close() }

func (r *relay) handle(w http.ResponseWriter, req *http.Request) {
	target := req.URL.Query().Get("u")
	out, err := http.NewRequest(req.Method, target, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if rg := req.Header.Get("Range"); rg != "" {
		out.Header.Set("Range", rg)
	}
	out.Header.Set("User-Agent", conply.DefaultBrowserUA)

	resp, err := r.client.Do(out)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "ETag", "Last-Modified"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
