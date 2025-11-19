# SAML 2.0 集成指南

本指南详细说明如何使用 TokenginX 实现完整的 SAML 2.0 单点登录系统。

## 概述

SAML (Security Assertion Markup Language) 2.0 是一个基于 XML 的开放标准,用于在身份提供者(IdP)和服务提供者(SP)之间交换认证和授权数据。TokenginX 提供高性能的 SAML 会话存储。

### 支持的绑定方式

- **HTTP-POST Binding** - 推荐,通过浏览器 POST 传输
- **HTTP-Redirect Binding** - 通过 URL 重定向传输
- **Artifact Binding** - 通过引用传输,安全性高

### 支持的流程

- **SP-Initiated SSO** - 服务提供者发起
- **IdP-Initiated SSO** - 身份提供者发起
- **Single Logout (SLO)** - 单点登出

## 数据结构设计

### SAML 会话存储

```
Key: saml:session:{session_index}
Value: {
  "session_index": "s2f5a8b3c1d4e9f7a6b5c4d3e2f1a0b9",
  "name_id": "user@example.com",
  "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
  "issuer": "https://idp.example.com",
  "audience": "https://sp.example.com",
  "assertion": "PFNhbWw6QXNzZXJ0aW9uLi4u",
  "attributes": {
    "email": "user@example.com",
    "displayName": "John Doe",
    "department": "Engineering",
    "roles": ["admin", "developer"]
  },
  "authn_instant": 1700000000,
  "session_not_on_or_after": 1700028800,
  "created_at": 1700000000
}
TTL: 28800 seconds (8 hours)
```

### Artifact 存储 (Artifact Binding)

```
Key: saml:artifact:{artifact_id}
Value: {
  "saml_response": "PFNhbWw6UmVzcG9uc2UuLi4=",
  "relay_state": "target_url",
  "created_at": 1700000000
}
TTL: 300 seconds (5 minutes,一次性使用)
```

### Logout Request 存储

```
Key: saml:logout:{request_id}
Value: {
  "request_id": "lr_abc123",
  "name_id": "user@example.com",
  "session_index": "s2f5a8b3c1d4e9f7a6b5c4d3e2f1a0b9",
  "issuer": "https://sp.example.com",
  "created_at": 1700000000
}
TTL: 300 seconds (5 minutes)
```

## SP-Initiated SSO 实现

### 1. 生成 SAML AuthnRequest

```go
import (
    "github.com/crewjam/saml"
    "github.com/crewjam/saml/samlsp"
)

func handleSPInitiatedSSO(w http.ResponseWriter, r *http.Request) {
    // 获取 SP 元数据
    sp := getSAMLServiceProvider()

    // 生成 AuthnRequest
    authnRequest := sp.MakeAuthenticationRequest(
        sp.GetSSOBindingLocation(saml.HTTPRedirectBinding),
        saml.HTTPRedirectBinding,
        saml.HTTPPostBinding,
    )

    // 设置 ForceAuthn (可选)
    authnRequest.ForceAuthn = saml.NewBool(false)

    // 设置 NameID 格式
    authnRequest.NameIDPolicy = &saml.NameIDPolicy{
        Format:     saml.NewString("urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"),
        AllowCreate: saml.NewBool(true),
    }

    // 保存 RelayState (目标URL)
    relayState := r.URL.Query().Get("target")
    if relayState == "" {
        relayState = "/"
    }

    // 重定向到 IdP
    redirectURL := authnRequest.Redirect(relayState)
    http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}
```

### 2. 处理 SAML Response (ACS)

```go
func handleAssertionConsumerService(w http.ResponseWriter, r *http.Request) {
    // 解析 POST 的 SAML Response
    err := r.ParseForm()
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    samlResponseEncoded := r.PostForm.Get("SAMLResponse")
    relayState := r.PostForm.Get("RelayState")

    // 解码 Base64
    samlResponseXML, err := base64.StdEncoding.DecodeString(samlResponseEncoded)
    if err != nil {
        http.Error(w, "Invalid SAML Response", http.StatusBadRequest)
        return
    }

    // 解析和验证 SAML Response
    sp := getSAMLServiceProvider()
    assertion, err := sp.ParseResponse(r, []string{""})
    if err != nil {
        log.Printf("Failed to parse SAML response: %v", err)
        http.Error(w, "Invalid SAML Response", http.StatusForbidden)
        return
    }

    // 验证 Assertion 签名
    if err := sp.ValidateAssertion(assertion); err != nil {
        log.Printf("Invalid assertion: %v", err)
        http.Error(w, "Invalid Assertion", http.StatusForbidden)
        return
    }

    // 提取用户信息
    nameID := assertion.Subject.NameID.Value
    sessionIndex := ""
    if len(assertion.AuthnStatements) > 0 {
        sessionIndex = assertion.AuthnStatements[0].SessionIndex
    }

    // 提取属性
    attributes := extractAttributes(assertion)

    // 存储会话到 TokenginX
    sessionData := map[string]interface{}{
        "session_index":          sessionIndex,
        "name_id":                nameID,
        "name_id_format":         assertion.Subject.NameID.Format,
        "issuer":                 assertion.Issuer.Value,
        "audience":               sp.MetadataURL.String(),
        "assertion":              base64.StdEncoding.EncodeToString(samlResponseXML),
        "attributes":             attributes,
        "authn_instant":          assertion.AuthnStatements[0].AuthnInstant,
        "session_not_on_or_after": assertion.AuthnStatements[0].SessionNotOnOrAfter,
        "created_at":             time.Now().Unix(),
    }

    // 计算会话过期时间
    sessionTTL := calculateSessionTTL(assertion)

    // 保存到 TokenginX
    ctx := context.Background()
    key := fmt.Sprintf("saml:session:%s", sessionIndex)
    value, _ := json.Marshal(sessionData)

    err = redisClient.Set(ctx, key, value, sessionTTL).Err()
    if err != nil {
        http.Error(w, "Failed to create session", http.StatusInternalServerError)
        return
    }

    // 创建应用会话 Cookie
    sessionCookie := &http.Cookie{
        Name:     "saml_session",
        Value:    sessionIndex,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   int(sessionTTL.Seconds()),
    }
    http.SetCookie(w, sessionCookie)

    // 重定向到目标 URL
    targetURL := relayState
    if targetURL == "" {
        targetURL = "/"
    }
    http.Redirect(w, r, targetURL, http.StatusFound)
}

func extractAttributes(assertion *saml.Assertion) map[string]interface{} {
    attributes := make(map[string]interface{})

    for _, attrStatement := range assertion.AttributeStatements {
        for _, attr := range attrStatement.Attributes {
            name := attr.FriendlyName
            if name == "" {
                name = attr.Name
            }

            var values []string
            for _, value := range attr.Values {
                values = append(values, value.Value)
            }

            if len(values) == 1 {
                attributes[name] = values[0]
            } else {
                attributes[name] = values
            }
        }
    }

    return attributes
}

func calculateSessionTTL(assertion *saml.Assertion) time.Duration {
    // 使用 SessionNotOnOrAfter
    if len(assertion.AuthnStatements) > 0 &&
       assertion.AuthnStatements[0].SessionNotOnOrAfter != nil {
        expiresAt := *assertion.AuthnStatements[0].SessionNotOnOrAfter
        ttl := time.Until(expiresAt)
        if ttl > 0 {
            return ttl
        }
    }

    // 默认 8 小时
    return 8 * time.Hour
}
```

### 3. 会话验证中间件

```go
func SAMLAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 获取会话 Cookie
        cookie, err := r.Cookie("saml_session")
        if err != nil {
            // 未登录,重定向到 SSO
            redirectToSSO(w, r)
            return
        }

        sessionIndex := cookie.Value

        // 从 TokenginX 获取会话
        ctx := context.Background()
        key := fmt.Sprintf("saml:session:%s", sessionIndex)

        sessionJSON, err := redisClient.Get(ctx, key).Result()
        if err == redis.Nil {
            // 会话已过期
            redirectToSSO(w, r)
            return
        }

        var sessionData map[string]interface{}
        json.Unmarshal([]byte(sessionJSON), &sessionData)

        // 将用户信息添加到请求上下文
        ctx = context.WithValue(r.Context(), "saml_user", sessionData)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func redirectToSSO(w http.ResponseWriter, r *http.Request) {
    targetURL := url.QueryEscape(r.URL.String())
    ssoURL := fmt.Sprintf("/saml/login?target=%s", targetURL)
    http.Redirect(w, r, ssoURL, http.StatusFound)
}
```

## IdP-Initiated SSO 实现

```go
func handleIdPInitiatedSSO(w http.ResponseWriter, r *http.Request) {
    // IdP 直接发送 SAML Response,无 InResponseTo 字段

    // 解析 SAML Response (与 SP-Initiated 类似)
    err := r.ParseForm()
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    samlResponseEncoded := r.PostForm.Get("SAMLResponse")

    // 解码和验证
    sp := getSAMLServiceProvider()
    assertion, err := sp.ParseResponse(r, []string{""})
    if err != nil {
        http.Error(w, "Invalid SAML Response", http.StatusForbidden)
        return
    }

    // IdP-Initiated 流程中 InResponseTo 应为空
    if assertion.InResponseTo != "" {
        http.Error(w, "Expected IdP-Initiated flow", http.StatusBadRequest)
        return
    }

    // 其余处理与 SP-Initiated 相同
    // ...
}
```

## Single Logout (SLO) 实现

### 1. SP 发起的登出

```go
func handleSPInitiatedLogout(w http.ResponseWriter, r *http.Request) {
    // 获取当前会话
    cookie, err := r.Cookie("saml_session")
    if err != nil {
        // 已经登出
        http.Redirect(w, r, "/", http.StatusFound)
        return
    }

    sessionIndex := cookie.Value

    // 从 TokenginX 获取会话信息
    ctx := context.Background()
    key := fmt.Sprintf("saml:session:%s", sessionIndex)

    sessionJSON, err := redisClient.Get(ctx, key).Result()
    if err != nil {
        // 会话不存在,清除 Cookie
        clearSessionCookie(w)
        http.Redirect(w, r, "/", http.StatusFound)
        return
    }

    var sessionData map[string]interface{}
    json.Unmarshal([]byte(sessionJSON), &sessionData)

    // 构造 LogoutRequest
    sp := getSAMLServiceProvider()
    logoutRequest := &saml.LogoutRequest{
        ID:           generateID(),
        IssueInstant: time.Now(),
        Issuer: &saml.Issuer{
            Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
            Value:  sp.MetadataURL.String(),
        },
        NameID: &saml.NameID{
            Format: sessionData["name_id_format"].(string),
            Value:  sessionData["name_id"].(string),
        },
        SessionIndex: &sessionData["session_index"].(string),
    }

    // 签名 LogoutRequest
    signedRequest, err := sp.SignLogoutRequest(logoutRequest)
    if err != nil {
        http.Error(w, "Failed to sign logout request", http.StatusInternalServerError)
        return
    }

    // 删除本地会话
    redisClient.Del(ctx, key)
    clearSessionCookie(w)

    // 重定向到 IdP 的 SLO 端点
    idpSLOURL := sp.GetSLOBindingLocation(saml.HTTPRedirectBinding)
    redirectURL := signedRequest.Redirect("", idpSLOURL)

    http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}
```

### 2. 处理 IdP 的 LogoutResponse

```go
func handleLogoutResponse(w http.ResponseWriter, r *http.Request) {
    samlResponseEncoded := r.URL.Query().Get("SAMLResponse")

    // 解码
    samlResponseXML, err := base64.StdEncoding.DecodeString(samlResponseEncoded)
    if err != nil {
        http.Error(w, "Invalid logout response", http.StatusBadRequest)
        return
    }

    // 解析 LogoutResponse
    var logoutResponse saml.LogoutResponse
    err = xml.Unmarshal(samlResponseXML, &logoutResponse)
    if err != nil {
        http.Error(w, "Failed to parse logout response", http.StatusBadRequest)
        return
    }

    // 验证签名
    sp := getSAMLServiceProvider()
    if err := sp.ValidateLogoutResponseSignature(&logoutResponse); err != nil {
        http.Error(w, "Invalid signature", http.StatusForbidden)
        return
    }

    // 检查状态
    if logoutResponse.Status.StatusCode.Value != saml.StatusSuccess {
        log.Printf("Logout failed: %s", logoutResponse.Status.StatusMessage)
    }

    // 重定向到登出成功页面
    http.Redirect(w, r, "/logout-success", http.StatusFound)
}
```

### 3. 处理 IdP 发起的登出

```go
func handleIdPInitiatedLogout(w http.ResponseWriter, r *http.Request) {
    samlRequestEncoded := r.URL.Query().Get("SAMLRequest")

    // 解码
    samlRequestXML, err := base64.StdEncoding.DecodeString(samlRequestEncoded)
    if err != nil {
        http.Error(w, "Invalid logout request", http.StatusBadRequest)
        return
    }

    // 解析 LogoutRequest
    var logoutRequest saml.LogoutRequest
    err = xml.Unmarshal(samlRequestXML, &logoutRequest)
    if err != nil {
        http.Error(w, "Failed to parse logout request", http.StatusBadRequest)
        return
    }

    // 验证签名
    sp := getSAMLServiceProvider()
    if err := sp.ValidateLogoutRequestSignature(&logoutRequest); err != nil {
        http.Error(w, "Invalid signature", http.StatusForbidden)
        return
    }

    // 删除会话
    ctx := context.Background()
    sessionIndex := *logoutRequest.SessionIndex
    key := fmt.Sprintf("saml:session:%s", sessionIndex)
    redisClient.Del(ctx, key)

    // 清除 Cookie
    clearSessionCookie(w)

    // 构造 LogoutResponse
    logoutResponse := &saml.LogoutResponse{
        ID:           generateID(),
        InResponseTo: logoutRequest.ID,
        IssueInstant: time.Now(),
        Issuer: &saml.Issuer{
            Value: sp.MetadataURL.String(),
        },
        Status: &saml.Status{
            StatusCode: &saml.StatusCode{
                Value: saml.StatusSuccess,
            },
        },
    }

    // 签名并重定向
    signedResponse, _ := sp.SignLogoutResponse(logoutResponse)
    idpSLOURL := sp.GetSLOBindingLocation(saml.HTTPRedirectBinding)
    redirectURL := signedResponse.Redirect("", idpSLOURL)

    http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}
```

## Artifact Binding 实现

### 1. 生成和存储 Artifact

```go
func handleArtifactBinding(w http.ResponseWriter, r *http.Request) {
    // 生成 SAML Response
    sp := getSAMLServiceProvider()
    samlResponse := generateSAMLResponse()

    // 序列化
    samlResponseXML, _ := xml.Marshal(samlResponse)
    samlResponseEncoded := base64.StdEncoding.EncodeToString(samlResponseXML)

    // 生成 Artifact
    artifact := generateArtifact()

    // 存储到 TokenginX (短时间)
    ctx := context.Background()
    artifactData := map[string]interface{}{
        "saml_response": samlResponseEncoded,
        "relay_state":   r.URL.Query().Get("RelayState"),
        "created_at":    time.Now().Unix(),
    }

    key := fmt.Sprintf("saml:artifact:%s", artifact)
    value, _ := json.Marshal(artifactData)
    redisClient.Set(ctx, key, value, 5*time.Minute)

    // 重定向到 SP,携带 Artifact
    redirectURL := fmt.Sprintf("%s?SAMLart=%s&RelayState=%s",
        sp.AcsURL.String(),
        url.QueryEscape(artifact),
        url.QueryEscape(artifactData["relay_state"].(string)))

    http.Redirect(w, r, redirectURL, http.StatusFound)
}

func generateArtifact() string {
    // SAML Artifact 格式: TypeCode + EndpointIndex + SourceID + MessageHandle
    // 简化实现
    bytes := make([]byte, 44) // TypeCode(2) + EndpointIndex(2) + SourceID(20) + MessageHandle(20)
    rand.Read(bytes[4:]) // 随机填充除 TypeCode 外的部分
    bytes[0] = 0x00
    bytes[1] = 0x04 // Type Code 0x0004 (SAML 2.0)

    return base64.StdEncoding.EncodeToString(bytes)
}
```

### 2. Artifact Resolution Service

```go
func handleArtifactResolution(w http.ResponseWriter, r *http.Request) {
    // 解析 SOAP 请求
    var artifactResolve saml.ArtifactResolve
    err := xml.NewDecoder(r.Body).Decode(&artifactResolve)
    if err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    artifact := artifactResolve.Artifact

    // 从 TokenginX 获取 SAML Response
    ctx := context.Background()
    key := fmt.Sprintf("saml:artifact:%s", artifact)

    artifactDataJSON, err := redisClient.Get(ctx, key).Result()
    if err == redis.Nil {
        // Artifact 不存在或已过期
        writeSOAPFault(w, "Artifact not found")
        return
    }

    var artifactData map[string]interface{}
    json.Unmarshal([]byte(artifactDataJSON), &artifactData)

    // 删除 Artifact (一次性使用)
    redisClient.Del(ctx, key)

    // 返回 SAML Response (SOAP 格式)
    samlResponseEncoded := artifactData["saml_response"].(string)
    samlResponseXML, _ := base64.StdEncoding.DecodeString(samlResponseEncoded)

    var samlResponse saml.Response
    xml.Unmarshal(samlResponseXML, &samlResponse)

    artifactResponse := &saml.ArtifactResponse{
        ID:           generateID(),
        InResponseTo: artifactResolve.ID,
        IssueInstant: time.Now(),
        Status: &saml.Status{
            StatusCode: &saml.StatusCode{Value: saml.StatusSuccess},
        },
        Response: &samlResponse,
    }

    // 序列化为 SOAP
    writeSOAPResponse(w, artifactResponse)
}
```

## 元数据配置

### SP 元数据

```xml
<?xml version="1.0"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
                     entityID="https://sp.example.com">
  <md:SPSSODescriptor
      AuthnRequestsSigned="true"
      WantAssertionsSigned="true"
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">

    <md:KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDEjCCAfqgAwIBAgI...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>

    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>

    <md:AssertionConsumerService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://sp.example.com/saml/acs"
        index="0" isDefault="true"/>

    <md:SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://sp.example.com/saml/slo"/>

  </md:SPSSODescriptor>
</md:EntityDescriptor>
```

## 安全最佳实践

### 1. 签名和加密

```go
// 始终验证签名
if err := sp.ValidateAssertion(assertion); err != nil {
    return errors.New("invalid signature")
}

// 要求 Assertion 加密(可选)
sp.AllowUnencryptedAssertion = false
```

### 2. 时间窗口验证

```go
// 验证 NotBefore 和 NotOnOrAfter
now := time.Now()
if assertion.Conditions.NotBefore.After(now) ||
   assertion.Conditions.NotOnOrAfter.Before(now) {
    return errors.New("assertion expired")
}

// 允许时钟偏移
clockSkew := 2 * time.Minute
```

### 3. Audience 验证

```go
// 验证 Audience 匹配
expectedAudience := sp.MetadataURL.String()
found := false
for _, audienceRestriction := range assertion.Conditions.AudienceRestrictions {
    for _, audience := range audienceRestriction.Audience {
        if audience.Value == expectedAudience {
            found = true
            break
        }
    }
}
if !found {
    return errors.New("audience mismatch")
}
```

### 4. InResponseTo 验证

```go
// SP-Initiated 流程必须验证 InResponseTo
expectedRequestID := getStoredRequestID(r)
if assertion.InResponseTo != expectedRequestID {
    return errors.New("InResponseTo mismatch")
}
```

## 客户端集成示例

### Spring Boot (Java)

```xml
<!-- pom.xml -->
<dependency>
    <groupId>org.springframework.security</groupId>
    <artifactId>spring-security-saml2-service-provider</artifactId>
</dependency>
```

```java
@Configuration
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .authorizeHttpRequests(authorize -> authorize
                .anyRequest().authenticated()
            )
            .saml2Login(Customizer.withDefaults())
            .saml2Logout(Customizer.withDefaults());

        return http.build();
    }

    @Bean
    public RelyingPartyRegistrationRepository relyingPartyRegistrationRepository() {
        RelyingPartyRegistration registration = RelyingPartyRegistrations
            .fromMetadataLocation("https://idp.example.com/metadata")
            .registrationId("my-idp")
            .build();

        return new InMemoryRelyingPartyRegistrationRepository(registration);
    }
}
```

## 监控和调试

### 1. 日志记录

```go
log.Printf("SAML SSO: user=%s, session=%s, issuer=%s",
    nameID, sessionIndex, issuer)
```

### 2. Prometheus 指标

```go
var (
    samlLogins = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "saml_logins_total",
        Help: "Total SAML logins",
    })

    samlLoginDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "saml_login_duration_seconds",
        Help: "SAML login duration",
    })
)
```

## 下一步

- 查看 [OAuth 2.0 集成指南](./oauth.md)
- 查看 [CAS 集成指南](./cas.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
- 了解 [防重放攻击](../security/anti-replay.md)
