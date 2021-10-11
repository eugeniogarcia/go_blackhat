# https-proxy

## Codificación

Define varios parámetros de entrada que nos permiten configurar el proxy como _http_ o como _https_, elegir el certificado y la clave privada:

```go
func main() {
	//Parametros de entrada
	var crtPath string
	flag.StringVar(&crtPath, "crt", "gz.com.crt", "path to crt file")
	var keyPath string
	flag.StringVar(&keyPath, "key", "gz.com.key", "path to key file")
	var proto string
	flag.StringVar(&proto, "proto", "https", "Proxy protocol (http or https)")
	flag.Parse()
```

El handler que procesará las peticiones que le lleguen al proxy, será diferente dependiendo de si se trata de un proxy _http_ o _https_:

```go
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
```

Usamos `handleTunneling(w, r)` para el caso _https_, y `handleHTTP(w, r)` para el caso _http_. Por último arrancamos el servidor:

```go
	if proto == "http" {
		log.Fatal(server.ListenAndServe())
	} else {
		log.Fatal(server.ListenAndServeTLS(crtPath, keyPath))
	}
```

### Handler http

Usa para enviar la request al backend usando `http.DefaultTransport.RoundTrip(req)`:

```go
func handleHTTP(w http.ResponseWriter, req *http.Request) {
	//Usamos el DefaultTransport.RoundTrip para hacer la petición al backend
	resp, err := http.DefaultTransport.RoundTrip(req)
```

Copiamos la respuesta en el stream a nuestro cliente:

```go
defer resp.Body.Close()
copyHeader(w.Header(), resp.Header)
w.WriteHeader(resp.StatusCode)
io.Copy(w, resp.Body)
}
```

### Handler https

Se conecta usando tcp al backend

```go
func handleTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
```

Captura la conexión con el cliente:

```go
w.WriteHeader(http.StatusOK)
//Toma el control de la conexión con el cliente
hijacker, ok := w.(http.Hijacker)
client_conn, _, err := hijacker.Hijack()
```

Copia los datos del cliente al backend y del backend al cliente

```go
go transfer(dest_conn, client_conn)
go transfer(client_conn, dest_conn)
}
```
