# HUMA Authentication & Authorization

## Document Security Schemes

Define security schemes in OpenAPI configuration:

```go
config := huma.DefaultConfig("My API", "1.0.0")
config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
    "bearer": {
        Type:         "http",
        Scheme:       "bearer",
        BearerFormat: "JWT",
        Description:  "Bearer token authentication",
    },
    "oauth2": {
        Type: "oauth2",
        Flows: &huma.OAuthFlows{
            AuthorizationCode: &huma.OAuthFlow{
                AuthorizationURL: "https://example.com/oauth/authorize",
                TokenURL:         "https://example.com/oauth/token",
                Scopes: map[string]string{
                    "read:users":  "Read user data",
                    "write:users": "Create/update users",
                    "admin":       "Admin access",
                },
            },
        },
    },
    "apiKey": {
        Type: "apiKey",
        In:   "header",
        Name: "X-API-Key",
    },
    "basic": {
        Type:   "http",
        Scheme: "basic",
    },
}
```

## Apply Security to Operations

```go
huma.Register(api, huma.Operation{
    OperationID: "get-protected",
    Method:      http.MethodGet,
    Path:        "/protected",
    Security: []map[string][]string{
        {"bearer": {}},
    },
}, handler)

// With OAuth scopes
huma.Register(api, huma.Operation{
    OperationID: "create-user",
    Method:      http.MethodPost,
    Path:        "/users",
    Security: []map[string][]string{
        {"oauth2": {"write:users"}},
    },
}, handler)

// Multiple options (any one is acceptable)
huma.Register(api, huma.Operation{
    OperationID: "get-resource",
    Method:      http.MethodGet,
    Path:        "/resources/{id}",
    Security: []map[string][]string{
        {"bearer": {}},
        {"apiKey": {}},
    },
}, handler)
```

## JWT Authentication Middleware

```go
import (
    "github.com/lestrrat-go/jwx/v2/jwt"
    "github.com/lestrrat-go/jwx/v2/jwk"
)

type JWTMiddleware struct {
    api    huma.API
    keySet jwk.Set
}

func NewJWTMiddleware(api huma.API, jwksURL string) *JWTMiddleware {
    // Fetch JWKS
    keySet, err := jwk.Fetch(context.Background(), jwksURL)
    if err != nil {
        panic(err)
    }

    return &JWTMiddleware{
        api:    api,
        keySet: keySet,
    }
}

func (m *JWTMiddleware) Middleware(ctx huma.Context, next func(huma.Context)) {
    // Check if operation requires auth
    op := ctx.Operation()
    if op.Security == nil || len(op.Security) == 0 {
        next(ctx)
        return
    }

    // Extract token
    authHeader := ctx.Header("Authorization")
    if authHeader == "" {
        huma.WriteErr(m.api, ctx, http.StatusUnauthorized, "Missing authorization header")
        return
    }

    token := strings.TrimPrefix(authHeader, "Bearer ")
    if token == authHeader {
        huma.WriteErr(m.api, ctx, http.StatusUnauthorized, "Invalid authorization header format")
        return
    }

    // Parse and validate JWT
    parsed, err := jwt.ParseString(token,
        jwt.WithKeySet(m.keySet),
        jwt.WithValidate(true),
    )
    if err != nil {
        huma.WriteErr(m.api, ctx, http.StatusUnauthorized, "Invalid token")
        return
    }

    // Validate scopes if required
    requiredScopes := getRequiredScopes(op)
    if len(requiredScopes) > 0 {
        tokenScopes := getTokenScopes(parsed)
        if !hasRequiredScopes(tokenScopes, requiredScopes) {
            huma.WriteErr(m.api, ctx, http.StatusForbidden, "Insufficient permissions")
            return
        }
    }

    // Add user info to context
    ctx = huma.WithValue(ctx, "user_id", parsed.Subject())
    ctx = huma.WithValue(ctx, "scopes", getTokenScopes(parsed))

    next(ctx)
}

func getRequiredScopes(op *huma.Operation) []string {
    for _, sec := range op.Security {
        if scopes, ok := sec["oauth2"]; ok {
            return scopes
        }
        if scopes, ok := sec["bearer"]; ok {
            return scopes
        }
    }
    return nil
}

func getTokenScopes(token jwt.Token) []string {
    scope, ok := token.Get("scope")
    if !ok {
        return nil
    }

    switch v := scope.(type) {
    case string:
        return strings.Split(v, " ")
    case []interface{}:
        scopes := make([]string, len(v))
        for i, s := range v {
            scopes[i] = s.(string)
        }
        return scopes
    }

    return nil
}

func hasRequiredScopes(tokenScopes, requiredScopes []string) bool {
    scopeSet := make(map[string]bool)
    for _, s := range tokenScopes {
        scopeSet[s] = true
    }

    for _, required := range requiredScopes {
        if !scopeSet[required] {
            return false
        }
    }

    return true
}
```

## API Key Authentication

```go
func APIKeyMiddleware(api huma.API, validKeys map[string]bool) func(huma.Context, func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        op := ctx.Operation()
        if op.Security == nil || len(op.Security) == 0 {
            next(ctx)
            return
        }

        apiKey := ctx.Header("X-API-Key")
        if apiKey == "" {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Missing API key")
            return
        }

        if !validKeys[apiKey] {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid API key")
            return
        }

        next(ctx)
    }
}

// Usage
validKeys := map[string]bool{
    "key123": true,
    "key456": true,
}

api.UseMiddleware(APIKeyMiddleware(api, validKeys))
```

## Basic Authentication

```go
func BasicAuthMiddleware(api huma.API, users map[string]string) func(huma.Context, func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        op := ctx.Operation()
        if op.Security == nil || len(op.Security) == 0 {
            next(ctx)
            return
        }

        authHeader := ctx.Header("Authorization")
        if authHeader == "" {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Missing authorization header")
            return
        }

        // Parse Basic auth
        const prefix = "Basic "
        if !strings.HasPrefix(authHeader, prefix) {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid authorization header")
            return
        }

        decoded, err := base64.StdEncoding.DecodeString(authHeader[len(prefix):])
        if err != nil {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid authorization header")
            return
        }

        parts := strings.SplitN(string(decoded), ":", 2)
        if len(parts) != 2 {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid authorization header")
            return
        }

        username, password := parts[0], parts[1]

        expectedPassword, ok := users[username]
        if !ok || password != expectedPassword {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid credentials")
            return
        }

        ctx = huma.WithValue(ctx, "username", username)
        next(ctx)
    }
}
```

## Role-Based Access Control

```go
type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
    RoleGuest Role = "guest"
)

func RequireRole(requiredRole Role) func(huma.Context, func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        userRole := ctx.Context().Value("user_role")
        if userRole == nil {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Not authenticated")
            return
        }

        role := userRole.(Role)

        // Define role hierarchy
        hierarchy := map[Role]int{
            RoleGuest: 1,
            RoleUser:  2,
            RoleAdmin: 3,
        }

        if hierarchy[role] < hierarchy[requiredRole] {
            huma.WriteErr(api, ctx, http.StatusForbidden, "Insufficient permissions")
            return
        }

        next(ctx)
    }
}

// Usage
huma.Register(api, huma.Operation{
    OperationID: "delete-user",
    Method:      http.MethodDelete,
    Path:        "/users/{id}",
    Security:    []map[string][]string{{"bearer": {}}},
    Middlewares: huma.Middlewares{RequireRole(RoleAdmin)},
}, deleteUserHandler)
```

## Resource-Level Authorization

```go
func handler(ctx context.Context, input *DeleteItemInput) (*DeleteItemOutput, error) {
    // Get authenticated user
    userID := ctx.Value("user_id").(string)

    // Fetch resource
    item, err := db.GetItem(input.ID)
    if err == sql.ErrNoRows {
        return nil, huma.Error404NotFound("Item not found")
    }
    if err != nil {
        return nil, huma.Error500InternalServerError("Failed to fetch item")
    }

    // Check ownership
    if item.OwnerID != userID {
        // Check if user is admin
        userRole := ctx.Value("user_role").(Role)
        if userRole != RoleAdmin {
            return nil, huma.Error403Forbidden("You don't have permission to delete this item")
        }
    }

    // Proceed with deletion
    if err := db.DeleteItem(input.ID); err != nil {
        return nil, huma.Error500InternalServerError("Failed to delete item")
    }

    return &DeleteItemOutput{}, nil
}
```

## OAuth2 Client Credentials Flow

```go
type OAuth2Config struct {
    TokenURL     string
    ClientID     string
    ClientSecret string
}

type tokenResponse struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    ExpiresIn   int    `json:"expires_in"`
}

func GetClientCredentialsToken(cfg OAuth2Config) (string, error) {
    data := url.Values{}
    data.Set("grant_type", "client_credentials")

    req, err := http.NewRequest("POST", cfg.TokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return "", fmt.Errorf("token request failed: %d", resp.StatusCode)
    }

    var token tokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
        return "", err
    }

    return token.AccessToken, nil
}
```

## Custom Claims

```go
type CustomClaims struct {
    UserID   string   `json:"user_id"`
    Email    string   `json:"email"`
    Roles    []string `json:"roles"`
    TenantID string   `json:"tenant_id"`
}

func ExtractClaims(token jwt.Token) (*CustomClaims, error) {
    claims := &CustomClaims{}

    if userID, ok := token.Get("user_id"); ok {
        claims.UserID = userID.(string)
    }

    if email, ok := token.Get("email"); ok {
        claims.Email = email.(string)
    }

    if roles, ok := token.Get("roles"); ok {
        if roleList, ok := roles.([]interface{}); ok {
            claims.Roles = make([]string, len(roleList))
            for i, r := range roleList {
                claims.Roles[i] = r.(string)
            }
        }
    }

    if tenantID, ok := token.Get("tenant_id"); ok {
        claims.TenantID = tenantID.(string)
    }

    return claims, nil
}
```

## Multi-Tenant Authorization

```go
func TenantMiddleware(api huma.API) func(huma.Context, func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        // Extract tenant from subdomain or header
        tenantID := ctx.Header("X-Tenant-ID")
        if tenantID == "" {
            host := ctx.Host()
            parts := strings.Split(host, ".")
            if len(parts) > 0 {
                tenantID = parts[0]
            }
        }

        if tenantID == "" {
            huma.WriteErr(api, ctx, http.StatusBadRequest, "Tenant not specified")
            return
        }

        // Validate tenant
        if !isValidTenant(tenantID) {
            huma.WriteErr(api, ctx, http.StatusNotFound, "Tenant not found")
            return
        }

        ctx = huma.WithValue(ctx, "tenant_id", tenantID)
        next(ctx)
    }
}
```

## Best Practices

1. **Always validate tokens** - Don't trust client input
2. **Use HTTPS in production** - Required for bearer tokens
3. **Implement token refresh** - Don't rely on long-lived tokens
4. **Check permissions at resource level** - Not just authentication
5. **Log auth failures** - Monitor for attacks
6. **Use short token expiry** - Balance security and UX
7. **Validate scopes/roles** - Don't assume token presence = authorization
8. **Handle expired tokens gracefully** - Return proper error codes
