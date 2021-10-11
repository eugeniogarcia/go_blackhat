# Curl 

[Curl](https://curl.se/docs/sslcerts.html):

```sh
curl httpbin.org/ip
```

Curl mostrando las cabeceras que se intercambian:

```sh
curl httpbin.org/ip -I
```

Curl ignorando errores de certificados en el proxy:

```sh
curl httpbin.org/ip --insecure
```

`--insecure` es equivalente a usar el flag `-k`. También destacar que la validación de certificados SSL\TLS se hace de forma independiente con el servidor y el proxy, así que tenemos flags para el servidor y para el proxy. Si usamos un proxy:

```sh
curl -x "http://user:pwd@192.168.1.162:8080" "http://httpbin.org/ip"

{
  "origin": "192.168.1.162, 139.47.67.45"
}
```

Si queremos ignorar los errores de certificados en el proxy:

```sh
curl -x "https://user:pwd@192.168.1.162:443" "http://httpbin.org/ip" --proxy-insecure

{
  "origin": "192.168.1.162, 139.47.67.45"
}
```

Podemos especificar el certificado de la _CA_ usando `--cacert` y `--proxy-cacert` para el servidor y el proxy respectivamente:

```sh
egsmartin@CPX-OOP5YZG1U89:~/Downloads$ curl -x "https://user:pwd@192.168.1.162:443" "http://httpbin.org/ip" --proxy-cace
rt ./myCA.pem

{
  "origin": "192.168.1.162, 139.47.67.45"
}
```

