- `tcp-proxy` escucha en el puerto https con TLS activado. Cargamos el certificado y la clave privada. El certificado fue firmado por nuestra CA. Hemos cargado nuestra CA como __Trusted Root CA__ con Chrome - o Edge -, de modo que la máquina la reconoce. Cada petición que recibimos la enviamos a _gz.com:443_. Hemos creado en nuestro __hosts__ una entrada para _gz.com_

Implementamos en el `tcp-proxy` un _Reader_ y un _Writer_ para demostrar como podríamos hacerlo. Estos no hace más que escribir en el log el payload que reciben.


- `http-ssl` implementa un servidor https que escucha en _gz.com:443_.

- `http-servidor`. No tiene nada especial. Creamos un servidor http que usaremos con el siguiente proyecto, el cliente http

- `http-client`. Creamos un roundtrip, y se explica como personalizar el cliente (timeout, pool de conexiones, ...). Usa también señales del SSOO.

