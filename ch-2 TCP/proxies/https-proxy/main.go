package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	//Se conecta usando tcp al backend
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	//Toma el control de la conexión con el cliente
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	//Copia los datos del cliente al backend
	go transfer(dest_conn, client_conn)
	//Copia los datos del backend al cliente
	go transfer(client_conn, dest_conn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	//Usamos el DefaultTransport.RoundTrip para hacer la petición al backend
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	//Copia la cabecera de la respuesta
	copyHeader(w.Header(), resp.Header)
	//el codigo
	w.WriteHeader(resp.StatusCode)
	//y el cuerpo
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func main() {
	//Parametros de entrada
	var crtPath string
	flag.StringVar(&crtPath, "crt", "gz.com.crt", "path to crt file")
	var keyPath string
	flag.StringVar(&keyPath, "key", "gz.com.key", "path to key file")
	var proto string
	flag.StringVar(&proto, "proto", "https", "Proxy protocol (http or https)")
	flag.Parse()

	if proto != "http" && proto != "https" {
		log.Fatal("Protocol must be either http or https")
	}

	//Configura el servidor http/https
	server := &http.Server{
		//Addr: ":8080",
		Addr: ":443",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//Si se trata de un CONNECT se esta iniciando una conexión TLS/SSL
			if r.Method == http.MethodConnect {
				handleTunneling(w, r) //handler para el caso https
			} else {
				handleHTTP(w, r) //handler para el caso http
			}
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	//Inicia el proxy http o https
	if proto == "http" {
		log.Fatal(server.ListenAndServe())
	} else {
		log.Fatal(server.ListenAndServeTLS(crtPath, keyPath))
	}
}
