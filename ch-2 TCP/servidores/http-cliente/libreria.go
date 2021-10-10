package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
)

type cacheTransport struct {
	data              map[string]string //Donde almacenamos los datos de la cache
	mu                sync.RWMutex      // Para serializar el acceso a la cache
	originalTransport http.RoundTripper
}

//Crea un roundtrip
func newTransport() *cacheTransport {
	return &cacheTransport{
		data:              make(map[string]string), //Crea la cache
		originalTransport: http.DefaultTransport,   //Usa un DefaultTransport
	}
}

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

//Crea una respuesta a partir de la información de la cache
func cachedResponse(b []byte, r *http.Request) (*http.Response, error) {
	//Crea un Buffer
	buf := bytes.NewBuffer(b)
	//Crea la respuesta con el Buffer que hemos creado
	return http.ReadResponse(bufio.NewReader(buf), r)
}

func (c *cacheTransport) RoundTrip(r *http.Request) (*http.Response, error) {

	// Check if we have the response cached..
	// If yes, we don't have to hit the server
	// We just return it as is from the cache store.
	if val, err := c.Get(r); err == nil {
		fmt.Println("Fetching the response from the cache")
		//Construimos la respuesta con el contenido de la cache
		return cachedResponse([]byte(val), r)
	}

	// Ok, we don't have the response cached, the store was probably cleared.
	// Make the request to the server.
	resp, err := c.originalTransport.RoundTrip(r)

	if err != nil {
		return nil, err
	}

	// Get the body of the response so we can save it in the cache for the next request.
	buf, err := httputil.DumpResponse(resp, true)

	if err != nil {
		return nil, err
	}

	// Saving it to the cache store
	c.Set(r, string(buf))

	fmt.Println("Fetching the data from the real source")
	return resp, nil
}
