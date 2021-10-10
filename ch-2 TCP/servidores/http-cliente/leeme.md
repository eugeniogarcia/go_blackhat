# Opciones para configurar el cliente http

## Timeout

The HTTP client does not contain the request timeout setting by default. If you are using http.Get(URL) or &Client{} that uses the http.DefaultClient. DefaultClient has not timeout setting; it comes with no timeout. Suppose the Rest API where you are making the request is broken, not sending the response back that keeps the connection open. More requests came, and open connection count will increase, Increasing server resources utilization, resulting in crashing your server when resource limits are reached.

We can specify the timeout in http.Client according to the use case

```go
var httpClient = &http.Client{
  Timeout: time.Second * 10,
}
```

## Connection Pooling

By default, the Golang Http client performs the connection pooling. When the request completes, that connection remains open until the idle connection timeout (default is 90 seconds). If another request came, that uses the same established connection instead of creating a new connection, after the idle connection time, the connection will return to the pool. When we do not define a transport in the http.Client, it uses the default transport Go HTTP Transport.

The default configuration of the HTTP Transport:

```go
var DefaultTransport RoundTripper = &Transport{
    Proxy: ProxyFromEnvironment,
    DialContext: (&net.Dialer{
        Timeout:   30 * time.Second,
        KeepAlive: 30 * time.Second,
    }).DialContext,
    ForceAttemptHTTP2:     true,
    MaxIdleConns:          100,
    IdleConnTimeout:       90 * time.Second,
    TLSHandshakeTimeout:   10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
}

// DefaultMaxIdleConnsPerHost is the default value of Transport's
// MaxIdleConnsPerHost.
const DefaultMaxIdleConnsPerHost = 2
```

__MaxIdleConns__ is the connection pool size, and this is the maximum connection that can be open; its default value is 100 connections. There is problem with the default setting __DefaultMaxIdleConnsPerHost__ with value of 2 connection. __DefaultMaxIdleConnsPerHost__ is the number of connection can be allowed to open per host basic. It means that for any particular host, out of 100 connection from the connection pool only two connections will be allocated to that host.

When more than two requests arrive, it will process only two requests and other requests will wait for the connection to communicate with the host server and go in the TIME_WAIT state.

__Solution__: Don't use the _Default Transport_ and increase MaxIdleConnsPerHost:

```go
t := http.DefaultTransport.(*http.Transport).Clone()
t.MaxIdleConns = 100
t.MaxConnsPerHost = 100
t.MaxIdleConnsPerHost = 100

httpClient = &http.Client{
    Timeout:   10 * time.Second,
    Transport: t,
}
```

__Transport is an implementation of RoundTripper that supports HTTP, HTTPS, and HTTP proxies__ (for either HTTP or HTTPS with CONNECT). By default, __Transport caches connections for future re-use__. This may leave many open connections when accessing many hosts. This behavior can be managed using Transport's CloseIdleConnections method and the MaxIdleConnsPerHost and DisableKeepAlives fields.

Transports should be reused instead of created as needed. Transports are safe for concurrent use by multiple goroutines. A Transport is a low-level primitive for making HTTP and HTTPS requests. For high-level functionality, such as cookies and redirects, see Client.

__Transport uses HTTP/1.1 for HTTP URLs and either HTTP/1.1 or HTTP/2 for HTTPS URLs__, depending on whether the server supports HTTP/2, and how the Transport is configured. The DefaultTransport supports HTTP/2. To explicitly enable HTTP/2 on a transport, use golang.org/x/net/http2 and call ConfigureTransport. See the package docs for more about HTTP/2. Responses with status codes in the 1xx range are either handled automatically (100 expect-continue) or ignored. The one exception is HTTP status code 101 (Switching Protocols), which is considered a terminal status and returned by RoundTrip. To see the ignored 1xx responses, use the httptrace trace package's ClientTrace.Got1xxResponse.

__Transport only retries a request upon encountering a network error if the request is idempotent and either has no body or has its Request.GetBody defined. HTTP requests are considered idempotent if they have HTTP methods GET, HEAD, OPTIONS, or TRACE; or if their Header map contains an "Idempotency-Key" or "X-Idempotency-Key" entry__. If the idempotency key value is a zero-length slice, the request is treated as idempotent but the header is not sent on the wire.

## http.RoundTripper

the ability to execute a single HTTP transaction, obtaining the Response for a given Request. Basically, what this means is being able to hook into what happens between making an HTTP request and receiving a response. In lay man terms, it’s like middleware but for an http.Client. I say this since round tripping occurs before the request is actually sent.

Since http.RoundTripper is an interface. All you have to do to get this functionality is implement RoundTrip :

```go
type SomeClient struct {}

func (s *SomeClient) RoundTrip(r *http.Request)(*Response, error) {
//Something comes here...Maybe
}
```

### Implementación

#### Type

Vamos a implementar un roundtrip que implementa una cache. En la definición de nuestro roundtrip incluimos el almacen de la cache, _data_, y un RW mutex para serializar el acceso a la cache, _mu_:

```go
type cacheTransport struct {
	data              map[string]string
	mu                sync.RWMutex
	originalTransport http.RoundTripper
}
```

Para crear nuestro roundtrip usamos este helper:

```go
//Crea un roundtrip
func newTransport() *cacheTransport {
	return &cacheTransport{
		data:              make(map[string]string), //Crea la cache
		originalTransport: http.DefaultTransport,   //Usa un DefaultTransport
	}
}
```

En originalTransport tenemos otro rountrip, el _http.DefaultTransport_. 

#### Cache

Los métodos para implementar la cache son los siguientes:

```go
//Implementamos una cache
//La key es la URL
func cacheKey(r *http.Request) string {
	return r.URL.String()
}

//Comprueba si tenemos la key en la cache
func (c *cacheTransport) Get(r *http.Request) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if val, ok := c.data[cacheKey(r)]; ok {
		return val, nil
	}

	return "", errors.New("key not found in cache")
}

//Guarda el valor en la cache
func (c *cacheTransport) Set(r *http.Request, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[cacheKey(r)] = value
}

//Limpia la cache
func (c *cacheTransport) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]string)
	return nil
}
```

La cache usa como key la _URL_.Tenemos métodos para guardar, recuperar y limpiar la cache.

#### Roundtrip propiamente dicho

Este helper permitirá crear una respuesta a partir de un slice de bytes que tenemos en la cache:

```go
//Crea una respuesta a partir de la información de la cache
func cachedResponse(b []byte, r *http.Request) (*http.Response, error) {
	//Crea un Buffer
	buf := bytes.NewBuffer(b)
	//Crea la respuesta con el Buffer que hemos creado
	return http.ReadResponse(bufio.NewReader(buf), r)
}
```

Lo primero cuando recibimos una petición es mirar si tenemos el valor en la cache:

```go
func (c *cacheTransport) RoundTrip(r *http.Request) (*http.Response, error) {

	// Check if we have the response cached..
	// If yes, we don't have to hit the server
	// We just return it as is from the cache store.
	if val, err := c.Get(r); err == nil {
		fmt.Println("Fetching the response from the cache")
		//Construimos la respuesta con el contenido de la cache
		return cachedResponse([]byte(val), r)
	}
```

Sino esta, usamos el roundtrip por defecto:

```go
	// Ok, we don't have the response cached, the store was probably cleared.
	// Make the request to the server.
	resp, err := c.originalTransport.RoundTrip(r)
```

y guardamos el valor en la cache:

```go
	// Get the body of the response so we can save it in the cache for the next request.
	buf, err := httputil.DumpResponse(resp, true)

	if err != nil {
		return nil, err
	}

	// Saving it to the cache store
	c.Set(r, string(buf))
```

#### Utilizar el roundtrip

Creamos un cliente custom:

```go
//Creamos nuestro roundtrip
cachedTransport := newTransport()

//Creamos un cliente custom, con nuestro roundtrip y aprovechamos para poner un time-out
client := &http.Client{
	Transport: cachedTransport,
	Timeout:   time.Second * 5,
}
```

Creamos una request:

```go
//Creamos la request
req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", strings.NewReader(""))
```

Hacemos la petición:

```go
resp, err := client.Do(req)
```

## Otros. Señales

Creamos un canal para recibir señales:

```go
terminateChannel := make(chan os.Signal, 1)
```

Le decimos al SSOO que queremos capturar las señales `syscall.SIGTERM` y `syscall.SIGHUP`:

```go
signal.Notify(terminateChannel, syscall.SIGTERM, syscall.SIGHUP)
```

Cuando se reciba una de estas señales se publicará en el channel. Procesamos el channel:

```go
case <-terminateChannel:
	cacheClearTicker.Stop()
	reqTicker.Stop()
	return
```