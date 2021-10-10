package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type MiReader struct{ conexion net.Conn }

func (r *MiReader) Read(b []byte) (int, error) {
	n, err := r.conexion.Read(b)
	fmt.Print(string(b))
	return n, err
}

type MiWriter struct{ conexion net.Conn }

// Write writes data to Stdout.
func (w *MiWriter) Write(b []byte) (int, error) {
	n, err := w.conexion.Write(b)
	return n, err
}

func handle(src net.Conn) {
	var wg sync.WaitGroup
	var fuente *MiReader = &MiReader{conexion: src}

	conf := &tls.Config{
		//InsecureSkipVerify: true,
	}
	dst, err := tls.Dial("tcp", "www.elpais.com:443", conf)
	//dst, err := net.Dial("tcp", "gz.com:8080")
	if err != nil {
		log.Fatalln("Unable to connect to our unreachable host")
	}
	defer dst.Close()

	var destino *MiWriter = &MiWriter{conexion: dst}

	// Run in goroutine to prevent io.Copy from blocking
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Envio")
		for {
			tmp := make([]byte, 256) // using small tmo buffer for demonstrating
			_, err := fuente.Read(tmp)
			if err != nil {
				if err != io.EOF {
					log.Fatalln(err)
				}
				break
			}
			destino.Write(tmp)
		}
		log.Println("Termina Envio")
	}()
	wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Respuesta")
		for {
			tmp := make([]byte, 256) // using small tmo buffer for demonstrating
			n, err := dst.Read(tmp)
			if err != nil {
				if err != io.EOF {
					log.Fatalln(err)
				}
				break
			}
			log.Println("got", n, "bytes.")
			src.Write(tmp)
		}
		log.Println("Termina Respuesta")
	}()
	// Copy our destination's output back to our source
	wg.Wait()
}

func main() {
	log.SetFlags(log.Lshortfile)
	/*
		cer, err := tls.LoadX509KeyPair("gz.com.crt", "gz.com.key")
		if err != nil {
			log.Println(err)
			return
		}
		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		listener, err := tls.Listen("tcp", "gz.com:443", config)
	*/
	listener, err := net.Listen("tcp", "gz.com:8080")
	if err != nil {
		log.Println(err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			log.Fatalln("Unable to accept connection")
		}
		go handle(conn)
	}
}
