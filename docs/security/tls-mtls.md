# TLS/mTLS 配置指南

本指南详细说明如何为 TokenginX 配置 TLS 加密通信和 mTLS 双向认证。

## 概述

TokenginX 支持 TLS 1.2 和 TLS 1.3 加密通信,推荐在生产环境中强制使用 TLS 1.3。支持标准的 X.509 证书和国密 SM2 证书。

### 安全传输模式

- **TLS (单向认证)** - 服务器证书认证
- **mTLS (双向认证)** - 服务器和客户端证书互相认证(推荐)

## TLS 配置

### 1. 生成自签名证书(开发环境)

```bash
# 生成 CA 私钥
openssl genrsa -out ca-key.pem 4096

# 生成 CA 证书
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/CN=MyCA"

# 生成服务器私钥
openssl genrsa -out server-key.pem 4096

# 生成服务器证书签名请求
openssl req -new -key server-key.pem -out server.csr \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/CN=tokenginx.example.com"

# 创建扩展配置文件
cat > server-ext.cnf <<EOF
subjectAltName = DNS:tokenginx.example.com,DNS:*.tokenginx.example.com,IP:127.0.0.1
extendedKeyUsage = serverAuth
EOF

# 使用 CA 签发服务器证书
openssl x509 -req -in server.csr -CA ca-cert.pem -CAkey ca-key.pem \
  -CAcreateserial -out server-cert.pem -days 365 \
  -extfile server-ext.cnf

# 验证证书
openssl verify -CAfile ca-cert.pem server-cert.pem
```

### 2. TokenginX 服务器 TLS 配置

在 `config.yaml` 中:

```yaml
server:
  tcp:
    addr: "0.0.0.0:6380"
    tls:
      enabled: true
      cert_file: "/etc/tokenginx/certs/server-cert.pem"
      key_file: "/etc/tokenginx/certs/server-key.pem"
      ca_file: "/etc/tokenginx/certs/ca-cert.pem"
      min_version: "1.3"  # 强制 TLS 1.3
      cipher_suites:
        - TLS_AES_256_GCM_SHA384
        - TLS_CHACHA20_POLY1305_SHA256
      client_auth: "none"  # none | request | require

  grpc:
    addr: "0.0.0.0:9090"
    tls:
      enabled: true
      cert_file: "/etc/tokenginx/certs/server-cert.pem"
      key_file: "/etc/tokenginx/certs/server-key.pem"
      client_auth: "none"

  http:
    addr: "0.0.0.0:8443"
    tls:
      enabled: true
      cert_file: "/etc/tokenginx/certs/server-cert.pem"
      key_file: "/etc/tokenginx/certs/server-key.pem"
```

### 3. 客户端 TLS 连接

#### Go 客户端

```go
import (
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
)

func connectWithTLS() (*redis.Client, error) {
    // 加载 CA 证书
    caCert, err := ioutil.ReadFile("/path/to/ca-cert.pem")
    if err != nil {
        return nil, err
    }

    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    // TLS 配置
    tlsConfig := &tls.Config{
        RootCAs:    caCertPool,
        MinVersion: tls.VersionTLS13,
        ServerName: "tokenginx.example.com",
    }

    // 创建 Redis 客户端
    client := redis.NewClient(&redis.Options{
        Addr:      "tokenginx.example.com:6380",
        TLSConfig: tlsConfig,
    })

    return client, nil
}
```

#### Java 客户端

```java
import io.lettuce.core.RedisClient;
import io.lettuce.core.RedisURI;
import io.lettuce.core.SslOptions;

public RedisClient connectWithTLS() {
    SslOptions sslOptions = SslOptions.builder()
        .truststore(new File("/path/to/truststore.jks"))
        .build();

    RedisURI redisURI = RedisURI.builder()
        .withHost("tokenginx.example.com")
        .withPort(6380)
        .withSsl(true)
        .withVerifyPeer(true)
        .build();

    RedisClient client = RedisClient.create(redisURI);
    client.setOptions(ClientOptions.builder()
        .sslOptions(sslOptions)
        .build());

    return client;
}
```

## mTLS 双向认证配置

### 1. 生成客户端证书

```bash
# 生成客户端私钥
openssl genrsa -out client-key.pem 4096

# 生成客户端证书签名请求
openssl req -new -key client-key.pem -out client.csr \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/CN=client001"

# 创建客户端扩展配置
cat > client-ext.cnf <<EOF
extendedKeyUsage = clientAuth
EOF

# 使用 CA 签发客户端证书
openssl x509 -req -in client.csr -CA ca-cert.pem -CAkey ca-key.pem \
  -CAcreateserial -out client-cert.pem -days 365 \
  -extfile client-ext.cnf

# 验证证书
openssl verify -CAfile ca-cert.pem client-cert.pem
```

### 2. 服务器 mTLS 配置

在 `config.yaml` 中:

```yaml
server:
  tcp:
    tls:
      enabled: true
      cert_file: "/etc/tokenginx/certs/server-cert.pem"
      key_file: "/etc/tokenginx/certs/server-key.pem"
      ca_file: "/etc/tokenginx/certs/ca-cert.pem"
      client_auth: "require"  # 强制客户端证书认证
      verify_client_cert: true
```

Go 实现示例:

```go
import (
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
)

func setupMTLSServer() (*tls.Config, error) {
    // 加载服务器证书
    cert, err := tls.LoadX509KeyPair(
        "/etc/tokenginx/certs/server-cert.pem",
        "/etc/tokenginx/certs/server-key.pem",
    )
    if err != nil {
        return nil, err
    }

    // 加载 CA 证书(用于验证客户端)
    caCert, err := ioutil.ReadFile("/etc/tokenginx/certs/ca-cert.pem")
    if err != nil {
        return nil, err
    }

    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    // TLS 配置
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientCAs:    caCertPool,
        ClientAuth:   tls.RequireAndVerifyClientCert,
        MinVersion:   tls.VersionTLS13,
        CipherSuites: []uint16{
            tls.TLS_AES_256_GCM_SHA384,
            tls.TLS_CHACHA20_POLY1305_SHA256,
        },
    }

    return tlsConfig, nil
}
```

### 3. 客户端 mTLS 连接

#### Go 客户端

```go
func connectWithMTLS() (*redis.Client, error) {
    // 加载客户端证书
    cert, err := tls.LoadX509KeyPair(
        "/path/to/client-cert.pem",
        "/path/to/client-key.pem",
    )
    if err != nil {
        return nil, err
    }

    // 加载 CA 证书
    caCert, err := ioutil.ReadFile("/path/to/ca-cert.pem")
    if err != nil {
        return nil, err
    }

    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    // mTLS 配置
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caCertPool,
        MinVersion:   tls.VersionTLS13,
        ServerName:   "tokenginx.example.com",
    }

    // 创建 Redis 客户端
    client := redis.NewClient(&redis.Options{
        Addr:      "tokenginx.example.com:6380",
        TLSConfig: tlsConfig,
    })

    return client, nil
}
```

#### .NET 客户端

```csharp
using StackExchange.Redis;
using System.Net.Security;
using System.Security.Cryptography.X509Certificates;

var options = new ConfigurationOptions
{
    EndPoints = { "tokenginx.example.com:6380" },
    Ssl = true,
    SslHost = "tokenginx.example.com",
};

// 加载客户端证书
var clientCert = new X509Certificate2("client-cert.pfx", "password");
options.CertificateSelection += delegate { return clientCert; };

// 验证服务器证书
options.CertificateValidation += (sender, cert, chain, errors) =>
{
    // 自定义验证逻辑
    if (errors == SslPolicyErrors.None)
        return true;

    // 检查证书链
    var caCert = new X509Certificate2("ca-cert.pem");
    chain.ChainPolicy.ExtraStore.Add(caCert);
    chain.ChainPolicy.VerificationFlags = X509VerificationFlags.AllowUnknownCertificateAuthority;

    return chain.Build((X509Certificate2)cert);
};

var connection = ConnectionMultiplexer.Connect(options);
```

#### Java 客户端

```java
import javax.net.ssl.*;
import java.io.FileInputStream;
import java.security.KeyStore;

public SSLContext createMTLSContext() throws Exception {
    // 加载客户端证书
    KeyStore keyStore = KeyStore.getInstance("PKCS12");
    keyStore.load(new FileInputStream("client-cert.p12"), "password".toCharArray());

    KeyManagerFactory kmf = KeyManagerFactory.getInstance("SunX509");
    kmf.init(keyStore, "password".toCharArray());

    // 加载 CA 证书
    KeyStore trustStore = KeyStore.getInstance("JKS");
    trustStore.load(new FileInputStream("truststore.jks"), "password".toCharArray());

    TrustManagerFactory tmf = TrustManagerFactory.getInstance("SunX509");
    tmf.init(trustStore);

    // 创建 SSL 上下文
    SSLContext sslContext = SSLContext.getInstance("TLSv1.3");
    sslContext.init(kmf.getKeyManagers(), tmf.getTrustManagers(), null);

    return sslContext;
}
```

## 证书管理

### 1. 证书轮换

```bash
#!/bin/bash
# cert-rotation.sh

# 生成新证书
openssl genrsa -out server-key-new.pem 4096
openssl req -new -key server-key-new.pem -out server-new.csr \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=MyOrg/CN=tokenginx.example.com"
openssl x509 -req -in server-new.csr -CA ca-cert.pem -CAkey ca-key.pem \
  -CAcreateserial -out server-cert-new.pem -days 365 \
  -extfile server-ext.cnf

# 验证新证书
openssl verify -CAfile ca-cert.pem server-cert-new.pem

# 备份旧证书
cp server-cert.pem server-cert.pem.backup
cp server-key.pem server-key.pem.backup

# 部署新证书
cp server-cert-new.pem /etc/tokenginx/certs/server-cert.pem
cp server-key-new.pem /etc/tokenginx/certs/server-key.pem

# 重新加载 TokenginX (热更新)
curl -X POST http://localhost:8080/admin/reload-cert \
  -H "Authorization: Bearer admin-token"
```

### 2. 证书热更新

Go 实现示例:

```go
import (
    "crypto/tls"
    "sync"
)

type CertificateManager struct {
    certFile string
    keyFile  string
    cert     *tls.Certificate
    mu       sync.RWMutex
}

func NewCertificateManager(certFile, keyFile string) (*CertificateManager, error) {
    cm := &CertificateManager{
        certFile: certFile,
        keyFile:  keyFile,
    }

    if err := cm.loadCertificate(); err != nil {
        return nil, err
    }

    return cm, nil
}

func (cm *CertificateManager) loadCertificate() error {
    cert, err := tls.LoadX509KeyPair(cm.certFile, cm.keyFile)
    if err != nil {
        return err
    }

    cm.mu.Lock()
    cm.cert = &cert
    cm.mu.Unlock()

    return nil
}

func (cm *CertificateManager) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    return cm.cert, nil
}

func (cm *CertificateManager) ReloadCertificate() error {
    return cm.loadCertificate()
}

// 使用示例
tlsConfig := &tls.Config{
    GetCertificate: certManager.GetCertificate,
}
```

### 3. 证书监控

```go
import (
    "crypto/x509"
    "time"
)

func monitorCertificateExpiry(certFile string) {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()

    for range ticker.C {
        cert, err := loadCertificate(certFile)
        if err != nil {
            log.Error("Failed to load certificate", "error", err)
            continue
        }

        daysUntilExpiry := time.Until(cert.NotAfter).Hours() / 24

        if daysUntilExpiry < 30 {
            alertCertificateExpiringSoon(certFile, daysUntilExpiry)
        }
    }
}
```

## Let's Encrypt 集成

### 1. 使用 Certbot 自动获取证书

```bash
# 安装 Certbot
sudo apt-get install certbot

# 获取证书
sudo certbot certonly --standalone \
  -d tokenginx.example.com \
  -d api.tokenginx.example.com \
  --email admin@example.com \
  --agree-tos \
  --non-interactive

# 证书路径
# /etc/letsencrypt/live/tokenginx.example.com/fullchain.pem
# /etc/letsencrypt/live/tokenginx.example.com/privkey.pem
```

### 2. 自动续期

```bash
# 添加 cron 任务
sudo crontab -e

# 每天凌晨 2 点检查续期
0 2 * * * certbot renew --quiet --deploy-hook "curl -X POST http://localhost:8080/admin/reload-cert"
```

### 3. ACME 协议集成

使用 `golang.org/x/crypto/acme/autocert`:

```go
import "golang.org/x/crypto/acme/autocert"

func setupAutoCert() *autocert.Manager {
    m := &autocert.Manager{
        Prompt:      autocert.AcceptTOS,
        HostPolicy:  autocert.HostWhitelist("tokenginx.example.com"),
        Cache:       autocert.DirCache("/var/lib/tokenginx/certs"),
        Email:       "admin@example.com",
    }

    return m
}

// 使用
tlsConfig := &tls.Config{
    GetCertificate: m.GetCertificate,
}
```

## 证书吊销检查

### 1. CRL (证书吊销列表)

```go
import "crypto/x509"

func checkCRL(cert *x509.Certificate) error {
    // 获取 CRL
    crlURL := cert.CRLDistributionPoints[0]
    resp, err := http.Get(crlURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    crlData, _ := ioutil.ReadAll(resp.Body)
    crl, err := x509.ParseCRL(crlData)
    if err != nil {
        return err
    }

    // 检查证书是否被吊销
    for _, revokedCert := range crl.TBSCertList.RevokedCertificates {
        if revokedCert.SerialNumber.Cmp(cert.SerialNumber) == 0 {
            return errors.New("certificate revoked")
        }
    }

    return nil
}
```

### 2. OCSP (在线证书状态协议)

```go
import "golang.org/x/crypto/ocsp"

func checkOCSP(cert, issuer *x509.Certificate) error {
    // 构造 OCSP 请求
    ocspReq, err := ocsp.CreateRequest(cert, issuer, nil)
    if err != nil {
        return err
    }

    // 发送 OCSP 请求
    ocspURL := cert.OCSPServer[0]
    resp, err := http.Post(ocspURL, "application/ocsp-request", bytes.NewReader(ocspReq))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    ocspRespData, _ := ioutil.ReadAll(resp.Body)

    // 解析 OCSP 响应
    ocspResp, err := ocsp.ParseResponse(ocspRespData, issuer)
    if err != nil {
        return err
    }

    // 检查状态
    if ocspResp.Status == ocsp.Revoked {
        return errors.New("certificate revoked")
    }

    return nil
}
```

## 安全最佳实践

### 1. 密码套件选择

```yaml
# 推荐配置(TLS 1.3)
cipher_suites:
  - TLS_AES_256_GCM_SHA384
  - TLS_CHACHA20_POLY1305_SHA256

# 禁用不安全的密码套件
# - TLS_RSA_WITH_RC4_128_SHA (不安全)
# - TLS_RSA_WITH_3DES_EDE_CBC_SHA (不安全)
```

### 2. 最低 TLS 版本

```yaml
# 强制 TLS 1.3
min_version: "1.3"

# 最低 TLS 1.2(兼容性考虑)
min_version: "1.2"
```

### 3. 证书验证

```go
// 始终验证服务器证书
tlsConfig.InsecureSkipVerify = false // ❌ 不要设置为 true

// 验证服务器名称
tlsConfig.ServerName = "tokenginx.example.com"

// 验证证书链
tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
    // 自定义验证逻辑
    return nil
}
```

### 4. 密钥强度

- RSA: 最少 2048 位,推荐 4096 位
- ECDSA: 最少 P-256,推荐 P-384
- SM2: 256 位

### 5. 证书有效期

- 服务器证书: 不超过 397 天(13 个月)
- 客户端证书: 不超过 2 年
- CA 证书: 不超过 10 年

## 故障排查

### 1. 验证 TLS 连接

```bash
# 使用 openssl 测试
openssl s_client -connect tokenginx.example.com:6380 \
  -CAfile ca-cert.pem \
  -cert client-cert.pem \
  -key client-key.pem \
  -tls1_3

# 查看证书信息
openssl x509 -in server-cert.pem -text -noout

# 验证证书链
openssl verify -CAfile ca-cert.pem -untrusted intermediate-cert.pem server-cert.pem
```

### 2. 常见错误

**错误: "x509: certificate signed by unknown authority"**
```
原因: 客户端未信任 CA 证书
解决: 将 CA 证书添加到客户端信任库
```

**错误: "tls: bad certificate"**
```
原因: 客户端证书无效或未提供
解决: 检查客户端证书配置
```

**错误: "x509: certificate has expired"**
```
原因: 证书已过期
解决: 更新证书
```

## 性能优化

### 1. 会话复用

```go
tlsConfig := &tls.Config{
    ClientSessionCache: tls.NewLRUClientSessionCache(100),
}
```

### 2. OCSP Stapling

```go
tlsConfig := &tls.Config{
    OCSPStaple: ocspResponse, // 服务器端配置
}
```

### 3. 硬件加速

使用硬件安全模块(HSM)或 AES-NI 加速 TLS 操作。

## 下一步

- 查看 [国密支持](./gm-crypto.md)
- 了解 [防重放攻击](./anti-replay.md)
- 配置 [访问控制(ACL)](./acl.md)
