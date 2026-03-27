# Mutual TLS Certificate Generation Guide

This guide explains how to generate certificates for mTLS (mutual TLS) between microservices, using `account-service` and `message-service` as examples.

## 1. Overview of mTLS

mTLS ensures that both client and server authenticate each other using certificates. A single Certificate Authority (CA) signs all service certificates to establish trust.

```
           CA
            │
   ┌────────┴────────┐
   │                 │
account-service   message-service
```

Each service has its own certificate and key, and all services share the CA certificate.

## 2. Required Files per Service

| Role   | Required Files                       | Purpose                                                                                        |
| ------ | ------------------------------------ | ---------------------------------------------------------------------------------------------- |
| Server | `server.crt`, `server.key`, `ca.crt` | `server.crt`: server identity, `server.key`: private key, `ca.crt`: verify client certificates |
| Client | `client.crt`, `client.key`, `ca.crt` | `client.crt`: client identity, `client.key`: private key, `ca.crt`: verify server certificates |

> Each service uses the same certificate for both client and server roles if it sends and receives requests.

## 3. Generate the CA

```bash
# Generate CA private key
openssl genrsa -out ca.key 4096

# Generate self-signed CA certificate
openssl req -x509 -new -nodes \
  -key ca.key \
  -sha256 \
  -days 3650 \
  -subj "/CN=openchat-ca" \
  -out ca.crt
```

* `ca.key`: CA private key (keep it secret)
* `ca.crt`: CA certificate, distributed to all services

## 4. Generate Certificate for account-service

### 4.1 Generate the service key

```bash
openssl genrsa -out account-service.key 2048
```

### 4.2 Create a CSR

```bash
openssl req -new \
  -key account-service.key \
  -subj "/CN=account-service" \
  -out account-service.csr
```

* `CN=account-service` identifies the service

### 4.3 Create SAN configuration (account-ext.cnf)

```text
subjectAltName = @alt_names

[alt_names]
DNS.1 = account-service
DNS.2 = localhost
```

* `DNS.1`: service name in Docker/Kubernetes network
* `DNS.2`: local address for testing

### 4.4 Sign the certificate with CA

```bash
openssl x509 -req \
  -in account-service.csr \
  -CA ca.crt \
  -CAkey ca.key \
  -CAcreateserial \
  -out account-service.crt \
  -days 365 \
  -sha256 \
  -extfile account-ext.cnf
```

* `-extfile account-ext.cnf` adds SAN to the certificate
* Result: `account-service.crt`

## 5. Generate Certificate for message-service

### 5.1 Generate the key

```bash
openssl genrsa -out message-service.key 2048
```

### 5.2 Create CSR

```bash
openssl req -new \
  -key message-service.key \
  -subj "/CN=message-service" \
  -out message-service.csr
```

### 5.3 SAN configuration (message-ext.cnf)

```text
subjectAltName = @alt_names

[alt_names]
DNS.1 = message-service
DNS.2 = localhost
```

### 5.4 Sign the certificate with CA

```bash
openssl x509 -req \
  -in message-service.csr \
  -CA ca.crt \
  -CAkey ca.key \
  -CAcreateserial \
  -out message-service.crt \
  -days 365 \
  -sha256 \
  -extfile message-ext.cnf
```

* Result: `message-service.crt`

## 6. Place Files in Service Containers

### account-service

```
ca.crt
account-service.crt
account-service.key
```

### message-service

```
ca.crt
message-service.crt
message-service.key
```

## 7. How It Works

1. `message-service` requests `account-service`.
2. `account-service` verifies the client certificate against `ca.crt`.
3. `account-service` sends its certificate; `message-service` verifies it against `ca.crt`.
4. Connection is established securely.

This works in reverse as well, enabling bidirectional mTLS.

## 8. Key Points

1. One CA signs all service certificates.
2. One certificate and key per service is sufficient.
3. CN (Common Name) identifies the service.
4. SAN (Subject Alternative Name) is required for modern TLS.
5. SAN configuration is stored in `.cnf` files.

## 9. Docker Example

### account-service volume mount

```yaml
volumes:
  - ./certs/ca.crt:/etc/certs/ca.crt:ro
  - ./certs/account-service.crt:/etc/certs/server.crt:ro
  - ./certs/account-service.key:/etc/certs/server.key:ro
```

### message-service volume mount

```yaml
volumes:
  - ./certs/ca.crt:/etc/certs/ca.crt:ro
  - ./certs/message-service.crt:/etc/certs/client.crt:ro
  - ./certs/message-service.key:/etc/certs/client.key:ro
```

## 10. Summary

* CA verifies all service certificates.
* Each service uses its certificate for both client and server roles.
* CN identifies the service.
* SAN ensures TLS validation for service DNS names.
* `.cnf` files are used to specify SAN entries.

This setup allows secure, bidirectional mTLS communication between microservices.
