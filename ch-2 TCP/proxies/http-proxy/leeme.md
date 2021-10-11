# http-proxy

No soporta trafico https. La razón es la siguiente:

__https traffic is encrypted with the session key at the client (and server) side__. Therefore __the traffic in the different sessions is always different, even if they are transferring the same content__. In order to serve something from the cache a proxy should have decrypted the traffic. Being able to decrypt the intercepted https traffic means a successful attack on https. Sometimes this is possible (google for OpenSSL heartbleed). However in general it's not feasible. And in most cases it's considered illegal.

## Código

Implementamos un tipo, proxy, que implementa el interface handler, y que por lo tanto podemos usar en un servidor:

```go
//Define un tipo que implementa el interface handler, y que por lo tanto podemos usar en un servidor
type proxy struct {
}

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
```

Al crear el servidor usaremos este tipo:

```go
handler := &proxy{}
if err := http.ListenAndServeTLS(*addr, "gz.com.crt", "gz.com.key", handler); err != nil {
```

El http handler del proxy lo que hace es crear un cliente http con la configuración por defecto:

```go
//Crea un cliente http
client := &http.Client{}
```

Vamos a usar la misma request que "le llegó" al proxy para enviarla al backend. Naturalmente tenemos que hacer algunas modificaciones antes en la request:

- Quitamos hop headers de la petición que hacemos al backend:

```go
delHopHeaders(req.Header)
```

```go
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
```

- Fijamos el header `X-Forwarded-For` con la ip del cliente:

```go
if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
	appendHostToXForwardHeader(req.Header, clientIP)
}
```

```go
func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}
```

Con esto ya tenemos la request como queremos, y podemos hacer la llamada al backend:

```go
	resp, err := client.Do(req)
	if err != nil {
		//Establece en la respuesta un error http
		http.Error(wr, err.Error(), http.StatusInternalServerError)
		log.Println("ServeHTTP:", err)
		return
	}
	defer resp.Body.Close()
```

La respuesta que obtenemos del backend la "copiamos" al writer de forma que le llegue a nuestro cliente. Quitamos los hop-headers, y pasamos las cabeceras, http status code y el body:

```go
	delHopHeaders(resp.Header)

	//Copiamos header, http code y cuerpo de la respuesta
	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
```