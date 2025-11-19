# 国密算法支持

TokenginX 原生支持中国国家商用密码算法（国密算法），满足国产化和信息安全合规要求。

## 概述

国密算法是中国国家密码管理局制定的商用密码标准，包括：

- **SM2**：椭圆曲线公钥密码算法（替代 RSA/ECDSA）
- **SM3**：密码杂凑算法（替代 SHA-256）
- **SM4**：分组密码算法（替代 AES）
- **SM9**：标识密码算法（可选，用于基于身份的加密）

TokenginX 同时支持国密和国际商密双体系，可根据需求灵活切换或混合使用。

## 国密算法详解

### SM2 - 公钥密码算法

SM2 是基于椭圆曲线密码学（ECC）的公钥密码算法，用于：

- **数字签名**：身份认证、数据完整性验证
- **密钥交换**：安全地协商会话密钥
- **加密解密**：非对称加密

**特点**：
- 256 位密钥长度
- 安全强度优于 RSA-2048
- 运算速度比 RSA 快 5-10 倍
- 密钥和签名尺寸更小

**使用场景**：
- TLS/mTLS 证书
- 请求签名验证
- 敏感数据非对称加密

### SM3 - 密码杂凑算法

SM3 是密码哈希算法，用于：

- **数据摘要**：生成固定长度的数据指纹
- **完整性校验**：验证数据未被篡改
- **HMAC**：生成消息认证码

**特点**：
- 输出 256 位（32 字节）哈希值
- 抗碰撞性、单向性
- 性能与 SHA-256 相当

**使用场景**：
- 请求签名（HMAC-SM3）
- 密码哈希
- 数据完整性校验

### SM4 - 分组密码算法

SM4 是对称加密算法，用于：

- **数据加密**：保护数据机密性
- **批量加密**：高效加密大量数据

**特点**：
- 128 位密钥长度
- 128 位分组长度
- 支持多种工作模式（CBC、GCM、CTR 等）
- 性能与 AES-128 相当

**使用场景**：
- 内存数据加密
- 持久化数据加密
- TLS 会话加密

## 国密 TLS 支持

TokenginX 支持国密 TLS（TLCP），遵循 GM/T 0024-2014 标准。

### 国密 TLS 特性

**双证书模式**：
- **签名证书**：用于身份认证和数字签名
- **加密证书**：用于密钥交换和加密

**密码套件**：
- `ECDHE_SM4_GCM_SM3`：推荐使用，支持前向保密
- `ECC_SM4_GCM_SM3`：传统模式

**TLS 版本**：
- 支持 TLCP 1.1（国密 TLS 1.1）
- 支持国密 TLS 1.3（新标准）

### 配置国密 TLS

#### 生成国密证书

```bash
# 使用 gmssl 工具生成签名证书
gmssl ecparam -genkey -name SM2 -out sign_key.pem
gmssl req -new -key sign_key.pem -out sign_csr.pem -sm3 \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=YourOrg/CN=tokenginx.example.com"
gmssl x509 -req -in sign_csr.pem -signkey sign_key.pem \
  -out sign_cert.pem -days 365 -sm3

# 生成加密证书
gmssl ecparam -genkey -name SM2 -out enc_key.pem
gmssl req -new -key enc_key.pem -out enc_csr.pem -sm3 \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=YourOrg/CN=tokenginx.example.com"
gmssl x509 -req -in enc_csr.pem -signkey enc_key.pem \
  -out enc_cert.pem -days 365 -sm3
```

#### 服务器配置

```yaml
# config.yaml
security:
  # 加密模式：gm (国密), sm (商密), auto (自动), hybrid (混合)
  crypto_mode: "gm"

  tls:
    enabled: true
    version: "tlcp-1.1"  # tlcp-1.1 或 gm-tls-1.3

    # 国密签名证书（必需）
    gm_sign_cert: "/path/to/sign_cert.pem"
    gm_sign_key: "/path/to/sign_key.pem"

    # 国密加密证书（必需）
    gm_enc_cert: "/path/to/enc_cert.pem"
    gm_enc_key: "/path/to/enc_key.pem"

    # CA 证书（用于验证客户端证书）
    ca_file: "/path/to/ca.pem"

    # 客户端认证模式
    client_auth: "require"  # none | request | require

    # 密码套件（可选，默认使用安全套件）
    cipher_suites:
      - "ECDHE_SM4_GCM_SM3"
      - "ECC_SM4_GCM_SM3"
```

#### 客户端连接（Go）

```go
package main

import (
    "crypto/tls"
    "log"

    "github.com/tjfoc/gmsm/gmtls"
    "github.com/your-org/tokenginx/client"
)

func main() {
    // 加载客户端国密证书
    signCert, err := gmtls.LoadX509KeyPair(
        "client_sign_cert.pem",
        "client_sign_key.pem",
    )
    if err != nil {
        log.Fatal(err)
    }

    encCert, err := gmtls.LoadX509KeyPair(
        "client_enc_cert.pem",
        "client_enc_key.pem",
    )
    if err != nil {
        log.Fatal(err)
    }

    // 加载 CA 证书
    caCert, err := ioutil.ReadFile("ca.pem")
    if err != nil {
        log.Fatal(err)
    }
    caCertPool := gmtls.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    // 配置国密 TLS
    tlsConfig := &gmtls.Config{
        Certificates:       []gmtls.Certificate{signCert, encCert},
        RootCAs:            caCertPool,
        InsecureSkipVerify: false,
    }

    // 连接到 TokenginX
    client, err := client.NewGMTLSClient("localhost:6380", tlsConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 使用客户端
    err = client.Set("oauth:token:abc123", sessionData, 3600)
    if err != nil {
        log.Fatal(err)
    }
}
```

## 数据加密

### 内存数据加密

TokenginX 使用 SM4-GCM 加密内存中的敏感数据（会话令牌、密码等）。

**配置**：

```yaml
security:
  crypto_mode: "gm"

  encryption:
    enabled: true
    algorithm: "sm4-gcm"  # sm4-gcm 或 aes-256-gcm

    # 主密钥来源
    master_key_source: "env"  # env | kms | file

    # 密钥轮换周期（天）
    key_rotation_days: 90
```

**环境变量方式**：

```bash
# 设置主密钥（256位 / 32字节，hex 编码）
export TOKENGINX_MASTER_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

# 启动服务器
./tokenginx-server -config config.yaml
```

### 持久化数据加密

mmap 文件和 WAL 日志使用 SM4-GCM 透明加密。

**配置**：

```yaml
storage:
  enable_persistence: true
  data_dir: "/var/lib/tokenginx"

  persistence:
    # 加密配置
    encryption:
      enabled: true
      algorithm: "sm4-gcm"
      # 加密块大小（字节）
      block_size: 4096
```

### 字段级加密

对特定字段单独加密，提供更细粒度的保护。

**示例**：

```go
// 在应用层使用国密加密特定字段
import "github.com/tjfoc/gmsm/sm4"

func encryptSensitiveField(plaintext []byte, key []byte) ([]byte, error) {
    // 使用 SM4-GCM 加密
    ciphertext, err := sm4.Sm4GCM(key, plaintext, nil, true)
    if err != nil {
        return nil, err
    }
    return ciphertext, nil
}
```

## 请求签名（HMAC-SM3）

使用 SM3 算法进行请求签名，防止篡改和重放攻击。

### 签名流程

**客户端签名**：

```go
import (
    "crypto/hmac"
    "encoding/hex"
    "fmt"
    "time"

    "github.com/tjfoc/gmsm/sm3"
)

func signRequest(method, uri, body string, secretKey []byte) (timestamp, nonce, signature string) {
    // 1. 生成时间戳和 Nonce
    timestamp = fmt.Sprintf("%d", time.Now().Unix())
    nonce = generateNonce() // 生成随机 Nonce

    // 2. 构造签名字符串
    signString := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
        method, uri, timestamp, nonce, body)

    // 3. 计算 HMAC-SM3
    h := hmac.New(sm3.New, secretKey)
    h.Write([]byte(signString))
    signature = hex.EncodeToString(h.Sum(nil))

    return timestamp, nonce, signature
}

// 使用示例
func makeSignedRequest() {
    method := "POST"
    uri := "/api/v1/sessions"
    body := `{"key":"oauth:token:abc123","value":"...","ttl":3600}`
    secretKey := []byte("your-secret-key")

    timestamp, nonce, signature := signRequest(method, uri, body, secretKey)

    // 发送请求时携带签名信息
    req, _ := http.NewRequest(method, "https://tokenginx.example.com"+uri, strings.NewReader(body))
    req.Header.Set("X-Timestamp", timestamp)
    req.Header.Set("X-Nonce", nonce)
    req.Header.Set("X-Signature", signature)
    req.Header.Set("Content-Type", "application/json")

    // 发送请求...
}
```

**服务器配置**：

```yaml
security:
  crypto_mode: "gm"

  anti_replay:
    enabled: true
    # 使用 SM3 进行签名验证
    signature_algorithm: "hmac-sm3"
    window_seconds: 300
    nonce_cache_size: 100000
```

## 国密与商密混合模式

TokenginX 支持同时接受国密和商密连接，适用于过渡期或多样化环境。

### 混合模式配置

```yaml
security:
  # 混合模式：同时支持国密和商密
  crypto_mode: "hybrid"

  tls:
    enabled: true

    # 商密证书
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"

    # 国密证书
    gm_sign_cert: "/path/to/sm2_sign.pem"
    gm_sign_key: "/path/to/sm2_sign_key.pem"
    gm_enc_cert: "/path/to/sm2_enc.pem"
    gm_enc_key: "/path/to/sm2_enc_key.pem"

    # 支持的密码套件（国密 + 商密）
    cipher_suites:
      # 国密套件
      - "ECDHE_SM4_GCM_SM3"
      - "ECC_SM4_GCM_SM3"
      # 商密套件
      - "TLS_AES_256_GCM_SHA384"
      - "TLS_CHACHA20_POLY1305_SHA256"
      - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
```

### 客户端协商

客户端在 TLS 握手时协商使用国密或商密：

```go
// 客户端支持国密和商密，由服务器选择
config := &tls.Config{
    // ... 配置证书 ...
    CipherSuites: []uint16{
        // 优先使用国密
        gmtls.ECDHE_SM4_GCM_SM3,
        // 回退到商密
        tls.TLS_AES_256_GCM_SHA384,
        tls.TLS_CHACHA20_POLY1305_SHA256,
    },
}
```

## 自动模式

自动模式会根据客户端能力自动选择国密或商密。

```yaml
security:
  # 自动模式：根据客户端能力选择
  crypto_mode: "auto"

  tls:
    enabled: true
    # 配置国密和商密证书...
```

**工作原理**：
1. 客户端发起 TLS 握手，携带支持的密码套件
2. 服务器检查客户端是否支持国密套件
3. 如果支持国密，优先使用国密；否则使用商密
4. 完成 TLS 握手，建立加密连接

## 密钥管理

### 主密钥管理

TokenginX 支持多种主密钥管理方式：

#### 环境变量（适用于开发/测试）

```bash
export TOKENGINX_MASTER_KEY="hex-encoded-key"
```

#### 配置文件（适用于简单部署）

```yaml
security:
  encryption:
    master_key_source: "file"
    master_key_file: "/secure/path/master.key"
```

**master.key 文件内容**（hex 编码的 256 位密钥）：
```
0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

#### KMS 集成（推荐生产环境）

```yaml
security:
  encryption:
    master_key_source: "kms"
    kms:
      provider: "aliyun"  # aliyun | tencent | huawei | aws
      region: "cn-beijing"
      key_id: "your-kms-key-id"
      # 访问凭证
      access_key_id: "${KMS_ACCESS_KEY_ID}"
      access_key_secret: "${KMS_ACCESS_KEY_SECRET}"
```

### 密钥轮换

TokenginX 支持自动密钥轮换，提高安全性。

```yaml
security:
  encryption:
    enabled: true
    # 密钥轮换周期（天）
    key_rotation_days: 90
    # 保留旧密钥的数量（用于解密旧数据）
    key_retention_count: 3
```

**密钥轮换流程**：
1. 到达轮换周期时，生成新的数据加密密钥（DEK）
2. 使用主密钥（MEK）加密新 DEK
3. 新数据使用新 DEK 加密
4. 旧数据逐步重新加密（后台任务）
5. 旧 DEK 保留一定时间后删除

### 硬件安全模块（HSM）

企业版支持硬件国密卡，提供最高级别的密钥保护。

```yaml
security:
  encryption:
    master_key_source: "hsm"
    hsm:
      provider: "sansec"  # 三未信安等国密硬件厂商
      device_path: "/dev/tass0"
      key_label: "tokenginx-master-key"
      pin: "${HSM_PIN}"
```

## 合规性

### 国产化要求

TokenginX 国密支持满足以下国产化要求：

- **算法合规**：使用国家密码管理局认证的 SM2/SM3/SM4 算法
- **标准遵循**：遵循 GM/T 系列国密标准
- **密码卡支持**：支持国产密码卡（HSM）
- **全栈国密**：传输层、存储层、应用层全面使用国密

### 等级保护要求

支持网络安全等级保护 2.0 要求：

- **三级等保**：完整的国密 TLS + 数据加密 + 审计日志
- **二级等保**：基础的 TLS 加密 + 访问控制

## 性能影响

### 国密 vs 商密性能对比

基于 Intel Xeon 8 核测试：

| 算法 | QPS | P99 延迟 | 相对商密性能 |
|------|-----|----------|-------------|
| SM2 签名 | 25,000 ops/s | 0.8ms | ~80% (vs RSA-2048) |
| SM2 验签 | 15,000 ops/s | 1.2ms | ~70% (vs RSA-2048) |
| SM3 哈希 | 500 MB/s | 0.05ms | ~95% (vs SHA-256) |
| SM4-GCM | 800 MB/s | 0.08ms | ~90% (vs AES-256-GCM) |

**结论**：国密算法性能略低于商密（约 10-20%），但仍能满足高性能要求。

### 性能优化建议

1. **使用硬件加速**：国密硬件卡性能可达商密水平
2. **会话复用**：TLS 会话复用减少握手开销
3. **批量操作**：批量加密/解密提高吞吐量
4. **连接池**：复用 TLS 连接

## 故障排查

### 常见问题

#### 1. 证书格式错误

**问题**：`failed to load GM certificate: invalid certificate format`

**解决**：
- 确保证书是 PEM 格式
- 确保是 SM2 椭圆曲线证书（非 RSA）
- 检查证书和私钥是否匹配

```bash
# 查看证书信息
gmssl x509 -in cert.pem -text -noout
```

#### 2. 密码套件不匹配

**问题**：`TLS handshake failed: no cipher suite supported`

**解决**：
- 检查客户端和服务器的密码套件配置
- 确保客户端支持国密套件
- 检查 TLS 版本兼容性

#### 3. 密钥权限问题

**问题**：`failed to read master key: permission denied`

**解决**：
```bash
# 确保密钥文件权限正确
chmod 600 /path/to/master.key
chown tokenginx:tokenginx /path/to/master.key
```

### 调试模式

启用国密调试日志：

```yaml
logging:
  level: "debug"
  # 启用国密相关日志
  modules:
    - "gm.tls"
    - "gm.sm2"
    - "gm.sm3"
    - "gm.sm4"
```

## 参考资料

### 国密标准文档

- [GM/T 0003.1-2012 SM2 椭圆曲线公钥密码算法](http://www.gmbz.org.cn/)
- [GM/T 0004-2012 SM3 密码杂凑算法](http://www.gmbz.org.cn/)
- [GM/T 0002-2012 SM4 分组密码算法](http://www.gmbz.org.cn/)
- [GM/T 0024-2014 SSL VPN 技术规范](http://www.gmbz.org.cn/)

### 开源库

- [tjfoc/gmsm](https://github.com/tjfoc/gmsm) - 国密算法 Go 实现
- [GmSSL](https://github.com/guanzhi/GmSSL) - 国密工具集

### 相关文档

- [TLS/mTLS 配置](./tls-mtls.md)
- [防重放攻击](./anti-replay.md)
- [访问控制 (ACL)](./acl.md)

---

**注意**：国密证书需要从国家认可的 CA 机构申请，测试环境可使用自签名证书。
