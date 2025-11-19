# CAS 集成指南

本指南详细说明如何使用 TokenginX 实现 CAS (Central Authentication Service) 单点登录系统。

## 概述

CAS 是一个企业级单点登录协议,最初由耶鲁大学开发。它简单、安全、易于集成,在教育和企业领域广泛使用。TokenginX 提供高性能的 CAS 票据存储。

### 支持的 CAS 版本

- **CAS 1.0** - 基础协议
- **CAS 2.0** - 支持代理认证
- **CAS 3.0** - 增强功能

### 核心概念

- **TGT (Ticket Granting Ticket)** - 票据授予票据,存储用户会话
- **ST (Service Ticket)** - 服务票据,用于访问具体服务
- **PGT (Proxy Granting Ticket)** - 代理票据,用于代理认证
- **PT (Proxy Ticket)** - 代理服务票据

## 数据结构设计

### TGT 存储

```
Key: cas:tgt:{tgt_id}
Value: {
  "tgt_id": "TGT-1-abc123def456",
  "username": "john.doe",
  "authentication_time": 1700000000,
  "created_at": 1700000000,
  "last_used_at": 1700000000,
  "attributes": {
    "email": "john@example.com",
    "displayName": "John Doe",
    "department": "Engineering"
  },
  "services": ["https://app1.example.com", "https://app2.example.com"]
}
TTL: 28800 seconds (8 hours, TGT 生命周期)
```

### ST 存储

```
Key: cas:st:{st_id}
Value: {
  "st_id": "ST-1-xyz789",
  "tgt_id": "TGT-1-abc123def456",
  "service": "https://app.example.com",
  "username": "john.doe",
  "created_at": 1700000000,
  "expires_at": 1700000300,
  "is_used": false,
  "attributes": {
    "email": "john@example.com",
    "displayName": "John Doe"
  }
}
TTL: 300 seconds (5 minutes, ST 短时有效)
```

### PGT 存储 (CAS 2.0+)

```
Key: cas:pgt:{pgt_id}
Value: {
  "pgt_id": "PGT-1-pqr456",
  "tgt_id": "TGT-1-abc123def456",
  "username": "john.doe",
  "target_service": "https://backend.example.com",
  "created_at": 1700000000
}
TTL: 28800 seconds (与 TGT 相同)
```

### PT 存储 (CAS 2.0+)

```
Key: cas:pt:{pt_id}
Value: {
  "pt_id": "PT-1-mno789",
  "pgt_id": "PGT-1-pqr456",
  "service": "https://backend-api.example.com",
  "username": "john.doe",
  "created_at": 1700000000,
  "is_used": false
}
TTL: 300 seconds (5 minutes)
```

## CAS 认证流程实现

### 1. /login - 登录端点

```go
func handleLogin(w http.ResponseWriter, r *http.Request) {
    service := r.URL.Query().Get("service")

    // 检查是否已有 TGT (通过 Cookie)
    tgtCookie, err := r.Cookie("TGC") // Ticket Granting Cookie
    if err == nil {
        // 验证 TGT
        tgtID := tgtCookie.Value
        if isValidTGT(tgtID) {
            // 已登录,生成 ST 并重定向
            st := generateServiceTicket(tgtID, service)
            redirectURL := fmt.Sprintf("%s?ticket=%s", service, st)
            http.Redirect(w, r, redirectURL, http.StatusFound)
            return
        }
    }

    // 未登录,显示登录页面
    if r.Method == "GET" {
        // 显示登录表单
        showLoginForm(w, service)
        return
    }

    // POST: 处理登录请求
    username := r.FormValue("username")
    password := r.FormValue("password")

    // 验证用户凭据
    if !authenticateUser(username, password) {
        showLoginForm(w, service)
        return
    }

    // 创建 TGT
    tgtID := generateTGT(username)

    // 设置 TGC Cookie
    tgcCookie := &http.Cookie{
        Name:     "TGC",
        Value:    tgtID,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   28800, // 8 hours
    }
    http.SetCookie(w, tgcCookie)

    // 如果有 service 参数,生成 ST 并重定向
    if service != "" {
        st := generateServiceTicket(tgtID, service)
        redirectURL := fmt.Sprintf("%s?ticket=%s", service, st)
        http.Redirect(w, r, redirectURL, http.StatusFound)
    } else {
        // 无 service,显示登录成功页面
        w.Write([]byte("Login successful"))
    }
}

func generateTGT(username string) string {
    tgtID := fmt.Sprintf("TGT-1-%s", generateSecureToken(32))

    // 获取用户属性
    attributes := getUserAttributes(username)

    tgtData := map[string]interface{}{
        "tgt_id":              tgtID,
        "username":            username,
        "authentication_time": time.Now().Unix(),
        "created_at":          time.Now().Unix(),
        "last_used_at":        time.Now().Unix(),
        "attributes":          attributes,
        "services":            []string{},
    }

    // 存储到 TokenginX
    ctx := context.Background()
    key := fmt.Sprintf("cas:tgt:%s", tgtID)
    value, _ := json.Marshal(tgtData)

    redisClient.Set(ctx, key, value, 8*time.Hour)

    return tgtID
}

func generateServiceTicket(tgtID, service string) string {
    // 从 TokenginX 获取 TGT
    ctx := context.Background()
    tgtKey := fmt.Sprintf("cas:tgt:%s", tgtID)

    tgtJSON, err := redisClient.Get(ctx, tgtKey).Result()
    if err != nil {
        return ""
    }

    var tgtData map[string]interface{}
    json.Unmarshal([]byte(tgtJSON), &tgtData)

    // 生成 ST
    stID := fmt.Sprintf("ST-1-%s", generateSecureToken(32))

    stData := map[string]interface{}{
        "st_id":      stID,
        "tgt_id":     tgtID,
        "service":    service,
        "username":   tgtData["username"],
        "created_at": time.Now().Unix(),
        "expires_at": time.Now().Add(5 * time.Minute).Unix(),
        "is_used":    false,
        "attributes": tgtData["attributes"],
    }

    // 存储 ST
    stKey := fmt.Sprintf("cas:st:%s", stID)
    stValue, _ := json.Marshal(stData)
    redisClient.Set(ctx, stKey, stValue, 5*time.Minute)

    // 更新 TGT 的服务列表和最后使用时间
    services := tgtData["services"].([]interface{})
    services = append(services, service)
    tgtData["services"] = services
    tgtData["last_used_at"] = time.Now().Unix()

    updatedTGT, _ := json.Marshal(tgtData)
    redisClient.Set(ctx, tgtKey, updatedTGT, 8*time.Hour)

    return stID
}
```

### 2. /serviceValidate - ST 验证端点 (CAS 2.0)

```go
func handleServiceValidate(w http.ResponseWriter, r *http.Request) {
    ticket := r.URL.Query().Get("ticket")
    service := r.URL.Query().Get("service")
    pgtUrl := r.URL.Query().Get("pgtUrl") // 用于代理认证

    if ticket == "" || service == "" {
        writeValidationFailure(w, "INVALID_REQUEST", "Missing required parameters")
        return
    }

    // 从 TokenginX 获取 ST
    ctx := context.Background()
    stKey := fmt.Sprintf("cas:st:%s", ticket)

    stJSON, err := redisClient.Get(ctx, stKey).Result()
    if err == redis.Nil {
        writeValidationFailure(w, "INVALID_TICKET", "Ticket not found or expired")
        return
    }

    var stData map[string]interface{}
    json.Unmarshal([]byte(stJSON), &stData)

    // 验证 ST 是否已使用
    if stData["is_used"].(bool) {
        writeValidationFailure(w, "INVALID_TICKET", "Ticket already used")
        return
    }

    // 验证 service 匹配
    if stData["service"].(string) != service {
        writeValidationFailure(w, "INVALID_SERVICE", "Service mismatch")
        return
    }

    // 验证过期时间
    expiresAt := int64(stData["expires_at"].(float64))
    if time.Now().Unix() > expiresAt {
        redisClient.Del(ctx, stKey)
        writeValidationFailure(w, "INVALID_TICKET", "Ticket expired")
        return
    }

    // 标记 ST 为已使用
    stData["is_used"] = true
    updatedST, _ := json.Marshal(stData)
    redisClient.Set(ctx, stKey, updatedST, 1*time.Minute)

    // 处理 PGT (如果提供了 pgtUrl)
    var pgtIOU string
    if pgtUrl != "" {
        pgtIOU = handleProxyGrantingTicket(stData, pgtUrl)
    }

    // 返回验证成功响应
    writeValidationSuccess(w, stData, pgtIOU)
}

func writeValidationSuccess(w http.ResponseWriter, stData map[string]interface{}, pgtIOU string) {
    w.Header().Set("Content-Type", "application/xml")

    username := stData["username"].(string)
    attributes := stData["attributes"].(map[string]interface{})

    xml := fmt.Sprintf(`<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas">
    <cas:authenticationSuccess>
        <cas:user>%s</cas:user>`, username)

    if pgtIOU != "" {
        xml += fmt.Sprintf(`
        <cas:proxyGrantingTicket>%s</cas:proxyGrantingTicket>`, pgtIOU)
    }

    // 添加属性
    if len(attributes) > 0 {
        xml += `
        <cas:attributes>`
        for key, value := range attributes {
            xml += fmt.Sprintf(`
            <cas:%s>%v</cas:%s>`, key, value, key)
        }
        xml += `
        </cas:attributes>`
    }

    xml += `
    </cas:authenticationSuccess>
</cas:serviceResponse>`

    w.Write([]byte(xml))
}

func writeValidationFailure(w http.ResponseWriter, code, message string) {
    w.Header().Set("Content-Type", "application/xml")

    xml := fmt.Sprintf(`<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas">
    <cas:authenticationFailure code="%s">%s</cas:authenticationFailure>
</cas:serviceResponse>`, code, message)

    w.Write([]byte(xml))
}
```

### 3. /validate - ST 验证端点 (CAS 1.0)

```go
func handleValidate(w http.ResponseWriter, r *http.Request) {
    ticket := r.URL.Query().Get("ticket")
    service := r.URL.Query().Get("service")

    // CAS 1.0 简化验证
    ctx := context.Background()
    stKey := fmt.Sprintf("cas:st:%s", ticket)

    stJSON, err := redisClient.Get(ctx, stKey).Result()
    if err != nil || !isValidST(stJSON, service) {
        w.Write([]byte("no\n"))
        return
    }

    var stData map[string]interface{}
    json.Unmarshal([]byte(stJSON), &stData)

    username := stData["username"].(string)
    w.Write([]byte(fmt.Sprintf("yes\n%s\n", username)))

    // 标记为已使用
    redisClient.Del(ctx, stKey)
}
```

## 代理认证实现 (CAS 2.0)

### 1. 生成 PGT

```go
func handleProxyGrantingTicket(stData map[string]interface{}, pgtUrl string) string {
    // 生成 PGT 和 PGT IOU
    pgtID := fmt.Sprintf("PGT-1-%s", generateSecureToken(32))
    pgtIOU := fmt.Sprintf("PGTIOU-1-%s", generateSecureToken(32))

    tgtID := stData["tgt_id"].(string)
    username := stData["username"].(string)

    pgtData := map[string]interface{}{
        "pgt_id":         pgtID,
        "tgt_id":         tgtID,
        "username":       username,
        "target_service": pgtUrl,
        "created_at":     time.Now().Unix(),
    }

    // 存储 PGT
    ctx := context.Background()
    pgtKey := fmt.Sprintf("cas:pgt:%s", pgtID)
    pgtValue, _ := json.Marshal(pgtData)
    redisClient.Set(ctx, pgtKey, pgtValue, 8*time.Hour)

    // 回调 pgtUrl,传递 PGT
    callbackURL := fmt.Sprintf("%s?pgtId=%s&pgtIou=%s", pgtUrl, pgtID, pgtIOU)
    go func() {
        resp, err := http.Get(callbackURL)
        if err != nil || resp.StatusCode != 200 {
            // 回调失败,删除 PGT
            redisClient.Del(ctx, pgtKey)
        }
    }()

    return pgtIOU
}
```

### 2. /proxy - 代理票据端点

```go
func handleProxy(w http.ResponseWriter, r *http.Request) {
    pgt := r.URL.Query().Get("pgt")
    targetService := r.URL.Query().Get("targetService")

    // 验证 PGT
    ctx := context.Background()
    pgtKey := fmt.Sprintf("cas:pgt:%s", pgt)

    pgtJSON, err := redisClient.Get(ctx, pgtKey).Result()
    if err == redis.Nil {
        writeProxyFailure(w, "INVALID_TICKET", "PGT not found or expired")
        return
    }

    var pgtData map[string]interface{}
    json.Unmarshal([]byte(pgtJSON), &pgtData)

    // 生成 PT
    ptID := fmt.Sprintf("PT-1-%s", generateSecureToken(32))

    ptData := map[string]interface{}{
        "pt_id":      ptID,
        "pgt_id":     pgt,
        "service":    targetService,
        "username":   pgtData["username"],
        "created_at": time.Now().Unix(),
        "is_used":    false,
    }

    // 存储 PT
    ptKey := fmt.Sprintf("cas:pt:%s", ptID)
    ptValue, _ := json.Marshal(ptData)
    redisClient.Set(ctx, ptKey, ptValue, 5*time.Minute)

    // 返回 PT
    writeProxySuccess(w, ptID)
}

func writeProxySuccess(w http.ResponseWriter, proxyTicket string) {
    w.Header().Set("Content-Type", "application/xml")

    xml := fmt.Sprintf(`<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas">
    <cas:proxySuccess>
        <cas:proxyTicket>%s</cas:proxyTicket>
    </cas:proxySuccess>
</cas:serviceResponse>`, proxyTicket)

    w.Write([]byte(xml))
}

func writeProxyFailure(w http.ResponseWriter, code, message string) {
    w.Header().Set("Content-Type", "application/xml")

    xml := fmt.Sprintf(`<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas">
    <cas:proxyFailure code="%s">%s</cas:proxyFailure>
</cas:serviceResponse>`, code, message)

    w.Write([]byte(xml))
}
```

### 3. /proxyValidate - PT 验证端点

```go
func handleProxyValidate(w http.ResponseWriter, r *http.Request) {
    ticket := r.URL.Query().Get("ticket")
    service := r.URL.Query().Get("service")

    // PT 验证逻辑与 ST 类似
    ctx := context.Background()

    // 尝试 ST
    stKey := fmt.Sprintf("cas:st:%s", ticket)
    if stJSON, err := redisClient.Get(ctx, stKey).Result(); err == nil {
        handleSTValidation(w, stJSON, service)
        return
    }

    // 尝试 PT
    ptKey := fmt.Sprintf("cas:pt:%s", ticket)
    ptJSON, err := redisClient.Get(ctx, ptKey).Result()
    if err == redis.Nil {
        writeValidationFailure(w, "INVALID_TICKET", "Ticket not found")
        return
    }

    handlePTValidation(w, ptJSON, service)
}
```

## Single Sign-Out (SLO) 实现

### 1. /logout - 登出端点

```go
func handleLogout(w http.ResponseWriter, r *http.Request) {
    service := r.URL.Query().Get("service")

    // 获取 TGC Cookie
    tgcCookie, err := r.Cookie("TGC")
    if err != nil {
        // 已登出
        if service != "" {
            http.Redirect(w, r, service, http.StatusFound)
        } else {
            w.Write([]byte("You have been logged out"))
        }
        return
    }

    tgtID := tgcCookie.Value

    // 从 TokenginX 获取 TGT
    ctx := context.Background()
    tgtKey := fmt.Sprintf("cas:tgt:%s", tgtID)

    tgtJSON, err := redisClient.Get(ctx, tgtKey).Result()
    if err == nil {
        var tgtData map[string]interface{}
        json.Unmarshal([]byte(tgtJSON), &tgtData)

        // 发送 SLO 请求到所有已访问的服务
        if services, ok := tgtData["services"].([]interface{}); ok {
            for _, svc := range services {
                sendSLORequest(svc.(string), tgtID)
            }
        }

        // 删除 TGT
        redisClient.Del(ctx, tgtKey)
    }

    // 清除 TGC Cookie
    clearTGCCookie(w)

    // 重定向
    if service != "" {
        http.Redirect(w, r, service, http.StatusFound)
    } else {
        w.Write([]byte("You have been logged out"))
    }
}

func sendSLORequest(serviceURL, tgtID string) {
    // 构造 SAML LogoutRequest (CAS 3.0)
    logoutRequest := fmt.Sprintf(`<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol">
    <samlp:SessionIndex>%s</samlp:SessionIndex>
</samlp:LogoutRequest>`, tgtID)

    // POST 到服务的 SLO 端点
    http.Post(serviceURL+"/cas-logout", "application/xml", strings.NewReader(logoutRequest))
}

func clearTGCCookie(w http.ResponseWriter) {
    cookie := &http.Cookie{
        Name:     "TGC",
        Value:    "",
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        MaxAge:   -1,
    }
    http.SetCookie(w, cookie)
}
```

### 2. 客户端 SLO 处理

```go
// 服务端处理 CAS SLO 请求
func handleCASLogout(w http.ResponseWriter, r *http.Request) {
    var logoutRequest struct {
        SessionIndex string `xml:"SessionIndex"`
    }

    xml.NewDecoder(r.Body).Decode(&logoutRequest)

    tgtID := logoutRequest.SessionIndex

    // 清除本地与该 TGT 关联的会话
    clearLocalSession(tgtID)

    w.WriteHeader(http.StatusOK)
}
```

## 客户端集成示例

### Java 客户端

```xml
<!-- pom.xml -->
<dependency>
    <groupId>org.jasig.cas.client</groupId>
    <artifactId>cas-client-core</artifactId>
    <version>3.6.4</version>
</dependency>
```

```java
// web.xml
<filter>
    <filter-name>CAS Authentication Filter</filter-name>
    <filter-class>org.jasig.cas.client.authentication.AuthenticationFilter</filter-class>
    <init-param>
        <param-name>casServerLoginUrl</param-name>
        <param-value>https://cas.example.com/login</param-value>
    </init-param>
    <init-param>
        <param-name>serverName</param-name>
        <param-value>https://myapp.example.com</param-value>
    </init-param>
</filter>

<filter>
    <filter-name>CAS Validation Filter</filter-name>
    <filter-class>org.jasig.cas.client.validation.Cas20ProxyReceivingTicketValidationFilter</filter-class>
    <init-param>
        <param-name>casServerUrlPrefix</param-name>
        <param-value>https://cas.example.com</param-value>
    </init-param>
    <init-param>
        <param-name>serverName</param-name>
        <param-value>https://myapp.example.com</param-value>
    </init-param>
</filter>

<filter-mapping>
    <filter-name>CAS Authentication Filter</filter-name>
    <url-pattern>/*</url-pattern>
</filter-mapping>

<filter-mapping>
    <filter-name>CAS Validation Filter</filter-name>
    <url-pattern>/*</url-pattern>
</filter-mapping>
```

### PHP 客户端

```php
<?php
require_once 'CAS.php';

// 初始化 phpCAS
phpCAS::client(CAS_VERSION_2_0, 'cas.example.com', 443, '/cas');

// 设置 CA 证书路径
phpCAS::setCasServerCACert('/path/to/ca-cert.pem');

// 强制认证
phpCAS::forceAuthentication();

// 获取用户名
$username = phpCAS::getUser();

// 获取属性
$attributes = phpCAS::getAttributes();
$email = $attributes['email'];
```

### Python 客户端

```python
from flask import Flask, redirect, url_for, session
from flask_cas import CAS

app = Flask(__name__)
app.config['CAS_SERVER'] = 'https://cas.example.com'
app.config['CAS_AFTER_LOGIN'] = 'index'

cas = CAS(app)

@app.route('/')
@cas.login_required
def index():
    username = cas.username
    attributes = cas.attributes
    return f'Hello {username}'

@app.route('/logout')
def logout():
    return redirect(cas.logout())
```

## 安全最佳实践

### 1. HTTPS 强制

```go
// 所有 CAS 端点必须使用 HTTPS
if r.TLS == nil {
    http.Error(w, "HTTPS required", http.StatusForbidden)
    return
}
```

### 2. Service URL 白名单

```go
var allowedServices = []string{
    "https://app1.example.com",
    "https://app2.example.com",
}

func isAllowedService(service string) bool {
    for _, allowed := range allowedServices {
        if strings.HasPrefix(service, allowed) {
            return true
        }
    }
    return false
}
```

### 3. 票据一次性使用

```go
// ST 和 PT 只能验证一次
if stData["is_used"].(bool) {
    return errors.New("ticket already used")
}
```

### 4. 票据短时有效

```go
// ST: 5 分钟
// PT: 5 分钟
// TGT: 8 小时(可配置)
```

## 监控指标

```go
var (
    casLogins = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "cas_logins_total",
        Help: "Total CAS logins",
    })

    casTicketsIssued = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cas_tickets_issued_total",
            Help: "Total CAS tickets issued",
        },
        []string{"type"}, // ST, TGT, PT, PGT
    )

    casValidationDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "cas_validation_duration_seconds",
            Help: "CAS ticket validation duration",
        },
    )
)
```

## 下一步

- 查看 [OAuth 2.0 集成指南](./oauth.md)
- 查看 [SAML 2.0 集成指南](./saml.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
- 了解 [防重放攻击](../security/anti-replay.md)
