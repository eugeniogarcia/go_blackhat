package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

//Define un tipo que implementa el interface handler, y que por lo tanto podemos usar en un servidor
type proxy struct {
}

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, " ", req.Method, " ", req.URL)

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		msg := "unsupported protocal scheme " + req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	//Crea un cliente http, con la configuración por defecto
	client := &http.Client{}

	//http: Request.RequestURI can't be set in client requests.
	//http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""

	delHopHeaders(req.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}

	//Usa la misma request, manipulada, para hacer la petición al backend
	resp, err := client.Do(req)
	if err != nil {
		//Establece en la respuesta un error http
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		log.Println("ServeHTTP:", err)
		return
	}
	defer resp.Body.Close()

	log.Println(req.RemoteAddr, " ", resp.Status)

	delHopHeaders(resp.Header)

	//Copiamos header, http code y cuerpo de la respuesta
	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func main() {
	var addr = flag.String("addr", ":443", "The addr of the application.")
	//var addr = flag.String("addr", ":8080", "The addr of the application.")
	flag.Parse()

	handler := &proxy{}

	log.Println("Starting proxy server on", *addr)
	//Escucha en claro
	//if err := http.ListenAndServe(*addr, handler); err != nil {
	//Escucha con TLS
	if err := http.ListenAndServeTLS(*addr, "gz.com.crt", "gz.com.key", handler); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
