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
	fmt.Println("Lee")
	fmt.Println(string(b))
	return n, err
}

type MiWriter struct{ conexion net.Conn }

// Write writes data to Stdout.
func (w *MiWriter) Write(b []byte) (int, error) {
	n, err := w.conexion.Write(b)
	fmt.Println("Escribe")
	fmt.Println(string(b))
	return n, err
}

func handle(src net.Conn) {
	var wg sync.WaitGroup
	//Fuente
	var envia *MiReader = &MiReader{conexion: src}
	var recibe *MiWriter = &MiWriter{conexion: src}

	//Se conecta con TLS
	conf := &tls.Config{
		//Si el sitio tiene un certificado emitido por una CA no reconocida, por ejemplo, selfsigned,
		//InsecureSkipVerify: true,
	}
	log.Println("Inicio")
	dst, err := tls.Dial("tcp", "gz.com:443", conf)
	if err != nil {
		log.Fatalln("Unable to connect to our unreachable host")
	}
	defer dst.Close()

	// Run in goroutine to prevent io.Copy from blocking
	wg.Add(1)
	go func() {
		defer wg.Done()
		tmp := make([]byte, 256) // using small tmo buffer for demonstrating
		for {
			n, err := envia.Read(tmp[0:])
			if err != nil {
				if err != io.EOF {
					log.Fatalln(err)
				}
				break
			}
			if _, err := dst.Write(tmp[0:n]); err != nil {
				log.Fatalln("Unable to write data")
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		tmp := make([]byte, 256) // using small tmo buffer for demonstrating
		for {
			n, err := dst.Read(tmp[0:])
			if err != nil {
				if err != io.EOF {
					log.Fatalln(err)
				}
				break
			}
			recibe.Write(tmp[0:n])
		}
	}()
	// Copy our destination's output back to our source
	wg.Wait()
	log.Println("Fin")
}

func main() {
	log.SetFlags(log.Lshortfile)

	//Carga el certificado y la clave privada
	cer, err := tls.LoadX509KeyPair("gz.com.crt", "gz.com.key")
	if err != nil {
		log.Println(err)
		return
	}
	//Escucha via TLS
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	listener, err := tls.Listen("tcp", ":443", config)
	//Escucha en claro
	//listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Println(err)
		return
	}
	defer listener.Close()

	for {
		//Acepta peticiones
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			log.Fatalln("Unable to accept connection")
		}
		go handle(conn)
	}
}
