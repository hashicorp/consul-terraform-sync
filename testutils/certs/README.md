# Test Certificates

## Consul Certificate
 `consul_cert.pem` is a self-signed certificate used for configuring Consul with TLS. The private key for the certificate is `consul_key.pem`. The same certificate is used by CTS as the CA authority for communicating with Consul on a TLS connection.

The DNS name for the certificate is `example.com` and the IP address is `127.0.0.1`. The certificate has a key usage restriction where it cannot be used for client authentication.

<details>
<summary>Example Consul Configuration Snippet</summary>

```
{
...
  "cert_file": "/path/to/consul_cert.pem",
  "key_file": "/path/to/consul_key.pem",
...
}
```
</details>

<details>
<summary>Example CTS Configuration Consul Block</summary>

```
consul {
  address = "127.0.0.1:34003"
  tls {
    enabled = true
    ca_cert = "/path/to/consul_cert.pem"
  }
}
```
</details>

## Localhost Root Certificates

`localhost_cert.pem` and `localhost_cert2.pem` are self-signed certificates for configuring TLS on the CTS API. The DNS names for the certificates are both `localhost`, and their respective private keys are `localhost_key.pem` and `localhost_cert2.pem`.

The certificates were generated via openssl with the following command:
```
$ openssl req -nodes -x509  -addext "subjectAltName = DNS:localhost" -subj '/CN=localhost' -keyout test_key.pem -out test_cert.pem -sha256 -days 3650 -newkey rsa:4096
```

> Note: If this command does not work for you because of `-addext`, you may need to install an updated version of OpenSSL or pass a configuration file with `-config` that has the Subject Alternative Name set.

## Leaf Certificates
`localhost_leaf_cert.pem` is a certificate issued and signed by `localhost_cert.pem`. If `localhost_cert.pem` is a trusted Certificate Authority, then `localhost_leaf_cert.pem` will be trusted as well. The DNS name for the certificate is `localhost`.

The certificate was generated via the following commands:
```
$ openssl req -nodes -newkey rsa:2048 -keyout localhost_leaf_key.pem -out localhost_leaf.csr 

$ openssl x509 -req -in localhost_leaf.csr -out localhost_leaf_cert.pem -CA localhost_cert.pem -CAkey localhost_key.pem -CAcreateserial -days 3650 -sha256 -extfile ext.cnf
```
Where `ext.cnf` has the following contents:
```
subjectAltName = DNS:localhost
```
