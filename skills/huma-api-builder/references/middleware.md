# HUMA Middleware Patterns

## Two Types of Middleware

### Router-Specific Middleware

Runs first, uses router's native middleware system:

```go
router := http.NewServeMux()

// Apply router middleware before creating API
router.Use(routerMiddleware)

api := humago.New(router, huma.DefaultConfig("My API", "1.0.0"))
```

### Router-Agnostic Middleware

Runs after router middleware, uses `huma.Context`:

```go
func MyMiddleware(ctx huma.Context, next func(huma.Context)) {
    // Pre-processing
    start := time.Now()

    // Call next handler
    next(ctx)

    // Post-processing
    duration := time.Since(start)
    log.Printf("Request took %v", duration)
}

// Apply to all operations
api.UseMiddleware(MyMiddleware)
```

## Common Middleware Patterns

### Logging Middleware

```go
func LoggingMiddleware(ctx huma.Context, next func(huma.Context)) {
    start := time.Now()
    method := ctx.Method()
    path := ctx.URL().Path

    next(ctx)

    status := ctx.Status()
    duration := time.Since(start)

    log.Printf("%s %s %d %v", method, path, status, duration)
}

api.UseMiddleware(LoggingMiddleware)
```

### Authentication Middleware

```go
func AuthMiddleware(ctx huma.Context, next func(huma.Context)) {
    token := strings.TrimPrefix(ctx.Header("Authorization"), "Bearer ")

    if token == "" {
        huma.WriteErr(api, ctx, http.StatusUnauthorized, "Missing authorization token")
        return
    }

    userID, err := validateToken(token)
    if err != nil {
        huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid token")
        return
    }

    // Add user to context
    ctx = huma.WithValue(ctx, "user_id", userID)

    next(ctx)
}

api.UseMiddleware(AuthMiddleware)
```

### CORS Middleware

```go
func CORSMiddleware(ctx huma.Context, next func(huma.Context)) {
    ctx.SetHeader("Access-Control-Allow-Origin", "*")
    ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")

    if ctx.Method() == http.MethodOptions {
        ctx.SetStatus(http.StatusNoContent)
        return
    }

    next(ctx)
}

api.UseMiddleware(CORSMiddleware)
```

### Rate Limiting Middleware

```go
func RateLimitMiddleware(limiter *rate.Limiter) func(huma.Context, func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        if !limiter.Allow() {
            huma.WriteErr(api, ctx, http.StatusTooManyRequests, "Rate limit exceeded")
            return
        }
        next(ctx)
    }
}

limiter := rate.NewLimiter(rate.Limit(10), 20) // 10 req/sec, burst 20
api.UseMiddleware(RateLimitMiddleware(limiter))
```

### Request ID Middleware

```go
func RequestIDMiddleware(ctx huma.Context, next func(huma.Context)) {
    requestID := ctx.Header("X-Request-ID")
    if requestID == "" {
        requestID = uuid.New().String()
    }

    ctx.SetHeader("X-Request-ID", requestID)
    ctx = huma.WithValue(ctx, "request_id", requestID)

    next(ctx)
}

api.UseMiddleware(RequestIDMiddleware)
```

### Timing Middleware

```go
func TimingMiddleware(ctx huma.Context, next func(huma.Context)) {
    start := time.Now()

    next(ctx)

    duration := time.Since(start)
    ctx.SetHeader("X-Response-Time", fmt.Sprintf("%dms", duration.Milliseconds()))
}

api.UseMiddleware(TimingMiddleware)
```

## Per-Operation Middleware

Apply middleware to specific operations:

```go
func AdminOnlyMiddleware(ctx huma.Context, next func(huma.Context)) {
    role := ctx.Context().Value("user_role").(string)

    if role != "admin" {
        huma.WriteErr(api, ctx, http.StatusForbidden, "Admin access required")
        return
    }

    next(ctx)
}

huma.Register(api, huma.Operation{
    OperationID: "delete-user",
    Method:      http.MethodDelete,
    Path:        "/users/{id}",
    Middlewares: huma.Middlewares{AdminOnlyMiddleware},
}, deleteUserHandler)
```

## Group Middleware

Apply middleware to groups of operations:

```go
adminGroup := huma.NewGroup(api, "/admin")
adminGroup.UseMiddleware(AdminOnlyMiddleware)

huma.Get(adminGroup, "/users", listUsersHandler)
huma.Delete(adminGroup, "/users/{id}", deleteUserHandler)
// Both operations have AdminOnlyMiddleware applied
```

## Context Values

### Setting Values

```go
func AuthMiddleware(ctx huma.Context, next func(huma.Context)) {
    userID := "user-123"
    ctx = huma.WithValue(ctx, "user_id", userID)
    next(ctx)
}
```

### Reading Values

```go
func handler(ctx context.Context, input *MyInput) (*MyOutput, error) {
    // Access from handler's context
    userID := ctx.Value("user_id").(string)

    return &MyOutput{}, nil
}
```

## Middleware Execution Order

```
1. Router-specific middleware (if any)
2. API-level router-agnostic middleware (in order added)
3. Group-level middleware (if operation is in a group)
4. Operation-level middleware (if specified)
5. Handler
6. (Back through middleware chain for post-processing)
```

Example:

```go
router := http.NewServeMux()
router.Use(routerLogging)  // 1. First

api := humago.New(router, config)
api.UseMiddleware(auth)     // 2. Second
api.UseMiddleware(rateLimit) // 3. Third

group := huma.NewGroup(api, "/api")
group.UseMiddleware(cors)   // 4. Fourth (for group operations)

huma.Register(api, huma.Operation{
    Path:        "/api/admin/users",
    Middlewares: huma.Middlewares{adminCheck}, // 5. Fifth
}, handler)  // 6. Finally
```

## Error Handling in Middleware

```go
func RecoveryMiddleware(api huma.API) func(huma.Context, func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Panic recovered: %v", r)
                huma.WriteErr(api, ctx, http.StatusInternalServerError, "Internal server error")
            }
        }()

        next(ctx)
    }
}

api.UseMiddleware(RecoveryMiddleware(api))
```

## Conditional Middleware

```go
func ConditionalMiddleware(ctx huma.Context, next func(huma.Context)) {
    // Check if operation requires authentication
    op := ctx.Operation()
    if op.Security != nil && len(op.Security) > 0 {
        // Apply auth check
        token := ctx.Header("Authorization")
        if token == "" {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")
            return
        }
    }

    next(ctx)
}
```

## Middleware Best Practices

1. **Order matters** - Apply middleware in logical order (logging → auth → rate limit → handler)
2. **Always call next()** - Unless intentionally stopping the chain (error, auth failure)
3. **Use context for state** - Pass data between middleware using `huma.WithValue`
4. **Handle panics** - Use recovery middleware at the top level
5. **Keep middleware focused** - Each middleware should do one thing well
6. **Consider performance** - Middleware runs on every request
7. **Use per-operation middleware sparingly** - Prefer API-level or group-level when possible
