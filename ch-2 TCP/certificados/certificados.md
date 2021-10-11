# Crea certificados

## Crea una Certificate Authority (CA)

### Genera la clave privada (*.key)

```ps
openssl genrsa -des3 -out myCA.key 2048
```

### Crea el certificado root (*.pem)

```ps
openssl req -x509 -new -nodes -key myCA.key -sha256 -days 825 -out myCA.pem
```

## Firma certificados con la CA

```ps
export NAME=gz.com
```

### Genera la clave privada (*.key)

```ps
openssl genrsa -out $NAME.key 2048
```

### Crea una petición de firma de la clave privada (*.csr)

```ps
openssl req -new -key $NAME.key -out $NAME.csr
```

### Crea el archivo de configuración donde se configuran las extensiones (*.ext)

```ps
cat >> $NAME.ext <<-EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names
[alt_names]
DNS.1 = $NAME # Be sure to include the domain name here because Common Name is not so commonly honoured by itself
DNS.2 = bar.$NAME # Optionally, add additional domains (I've added a subdomain here)
DNS.3 = localhost # Optionally, add additional domains (I've added a subdomain here)
IP.1 = 192.168.1.162 # Optionally, add an IP address (if the connection which you have planned requires it)
EOF
```

### Crea un certificado firmado por la CA  (*.crt)

```ps
openssl x509 -req -in $NAME.csr -CA myCA.pem -CAkey myCA.key -CAcreateserial \
-out $NAME.crt -days 825 -sha256 -extfile $NAME.ext
```

### Verificar el certificado

```ps
openssl verify -CAfile myCA.pem -verify_hostname localhost gz.com.crt
```
