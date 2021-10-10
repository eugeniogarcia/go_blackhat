# http-proxy

No soporta trafico https. La razón es la siguiente:

__https traffic is encrypted with the session key at the client (and server) side__. Therefore __the traffic in the different sessions is always different, even if they are transferring the same content__. In order to serve something from the cache a proxy should have decrypted the traffic. Being able to decrypt the intercepted https traffic means a successful attack on https. Sometimes this is possible (google for OpenSSL heartbleed). However in general it's not feasible. And in most cases it's considered illegal.

## Código

El http handler del proxy lo que hace es crear un cliente http:

```go
	//Crea un cliente http
	client := &http.Client{}
```

Quitamos hop headers de la petición que hacemos al backend:

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

Fijamos el header `X-Forwarded-For` con la ip del cliente:

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

y se hace la llamada al backend:

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

quitamos los hop-headers, y pasamos las cabeceras, http status code y el body:

```go
	delHopHeaders(resp.Header)

	//Copiamos header, http code y cuerpo de la respuesta
	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
```