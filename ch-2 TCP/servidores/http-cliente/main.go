package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {

	//Creamos nuestro roundtrip
	cachedTransport := newTransport()

	//Creamos un cliente custom, con nuestro roundtrip y aprovechamos para poner un time-out
	client := &http.Client{
		Transport: cachedTransport,
		Timeout:   time.Second * 5,
	}

	//Time to clear the cache store so we can make request to the original server
	cacheClearTicker := time.NewTicker(time.Second * 5)

	//Make a new request every second
	//This would help demonstrate if the response is actually coming from the real server or from the cache
	reqTicker := time.NewTicker(time.Second * 1)

	terminateChannel := make(chan os.Signal, 1)

	signal.Notify(terminateChannel, syscall.SIGTERM, syscall.SIGHUP)

	//Creamos la request
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080", strings.NewReader(""))

	if err != nil {
		log.Fatalf("An error occurred ... %v", err)
	}

	for {
		select {
		//Tick de la cache
		case <-cacheClearTicker.C:
			// Clear the cache so we can hit the original server
			cachedTransport.Clear()

		//Procesa una señal
		case <-terminateChannel:
			cacheClearTicker.Stop()
			reqTicker.Stop()
			return

		//Tick llamada
		case <-reqTicker.C:
			//Hace una llamada al servidor
			resp, err := client.Do(req)

			if err != nil {
				log.Printf("An error occurred.... %v", err)
				continue
			}

			buf, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				log.Printf("An error occurred.... %v", err)
				continue
			}

			fmt.Printf("The body of the response is \"%s\" \n\n", string(buf))
		}
	}
}
