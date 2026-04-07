---
name: huma-api-builder
description: Expert guidance for building REST APIs with HUMA v2 framework in Go. Use when creating new HUMA APIs, adding endpoints to existing HUMA projects, implementing validation/middleware/auth, structuring HUMA projects (clean architecture, monolith, microservices, flat), testing HUMA APIs, handling streaming/SSE, or troubleshooting HUMA-specific issues. Always uses Go 1.22+ standard library router (humago adapter).
---

# HUMA API Builder

Build production-ready REST APIs with HUMA v2 framework using Go 1.22+ standard library router.

## Quick Start

### Create New API

Choose architecture based on project needs:

**Simple/Flat (prototypes, learning, <5 endpoints):**
```bash
# Copy template
cp -r assets/simple/* myapi/
cd myapi && go mod tidy && go run main.go
```

**Clean Architecture (medium-large, long-term maintenance):**
```bash
# Copy template
cp -r assets/clean/* myapi/
cd myapi && go mod tidy && go run cmd/api/main.go
```

See [references/architecture.md](references/architecture.md) for all patterns: monolith, microservices, hexagonal, feature-based.

### Add Endpoint to Existing API

```go
// Define input/output
type GetUserInput struct {
    ID string `path:"id" format:"uuid"`
}

type GetUserOutput struct {
    Body struct {
        Username string `json:"username" example:"john_doe"`
        Email    string `json:"email" format:"email"`
    }
}

// Implement handler
func getUserHandler(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
    user, err := db.GetUser(input.ID)
    if err != nil {
        return nil, huma.Error404NotFound("User not found")
    }

    resp := &GetUserOutput{}
    resp.Body.Username = user.Username
    resp.Body.Email = user.Email
    return resp, nil
}

// Register endpoint
huma.Get(api, "/users/{id}", getUserHandler)
```

## Core Patterns

### Request Inputs

**Path parameters:**
```go
type Input struct {
    ID string `path:"id" format:"uuid"`
}
```

**Query parameters:**
```go
type Input struct {
    Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"10"`
    Offset int    `query:"offset" minimum:"0" default:"0"`
    Search string `query:"search" maxLength:"50"`
}
```

**Headers:**
```go
type Input struct {
    Auth  string `header:"Authorization" pattern:"^Bearer .+$"`
    Token string `header:"X-API-Key"`
}
```

**Request body:**
```go
type Input struct {
    Body struct {
        Name  string `json:"name" minLength:"1" maxLength:"100"`
        Email string `json:"email" format:"email"`
        Age   int    `json:"age" minimum:"0" maximum:"150"`
    }
}
```

**Optional fields:**
```go
type Input struct {
    Required string  `json:"required"`                 // Required by default
    Optional *string `json:"optional,omitempty"`       // Optional (pointer + omitempty)
    OptInt   int     `json:"opt_int,omitzero"`         // Optional with zero allowed
}
```

For comprehensive validation patterns, see [references/validation.md](references/validation.md).

### Response Outputs

**Simple response:**
```go
type Output struct {
    Body struct {
        Message string `json:"message"`
    }
}
```

**With status code:**
```go
type Output struct {
    Status int `header:"-"` // Dynamic status
    Body   User
}

// In handler
resp := &Output{Status: 201, Body: user}
```

**With headers:**
```go
type Output struct {
    ContentType string    `header:"Content-Type"`
    ETag        string    `header:"ETag"`
    LastMod     time.Time `header:"Last-Modified"`
    Body        Resource
}
```

**No content (204):**
```go
func deleteHandler(ctx context.Context, input *DeleteInput) (*struct{}, error) {
    // Delete logic
    return &struct{}{}, nil // Returns 204
}
```

### Error Handling

**Built-in errors (RFC 9457):**
```go
// Not found
return nil, huma.Error404NotFound("User not found")

// Validation
return nil, huma.Error422UnprocessableEntity("Invalid email")

// Conflict
return nil, huma.Error409Conflict("Username already taken")

// Internal
return nil, huma.Error500InternalServerError("Database error")
```

**Detailed errors:**
```go
return nil, huma.Error422UnprocessableEntity(
    "Validation failed",
    &huma.ErrorDetail{
        Message:  "Must be at least 8 characters",
        Location: "body.password",
        Value:    len(input.Body.Password),
    },
)
```

**Multiple errors:**
```go
var errs []*huma.ErrorDetail
if input.Body.Username == "" {
    errs = append(errs, &huma.ErrorDetail{
        Message: "Username required", Location: "body.username",
    })
}
if len(errs) > 0 {
    return nil, huma.Error422UnprocessableEntity("Validation failed", errs...)
}
```

See [references/errors.md](references/errors.md) for comprehensive error handling patterns.

### Middleware

**API-level middleware (all operations):**
```go
func LoggingMiddleware(ctx huma.Context, next func(huma.Context)) {
    start := time.Now()
    next(ctx)
    log.Printf("%s %s %d %v", ctx.Method(), ctx.URL().Path, ctx.Status(), time.Since(start))
}

api.UseMiddleware(LoggingMiddleware)
```

**Per-operation middleware:**
```go
huma.Register(api, huma.Operation{
    OperationID: "admin-only",
    Method:      http.MethodDelete,
    Path:        "/admin/users/{id}",
    Middlewares: huma.Middlewares{AdminOnlyMiddleware},
}, handler)
```

**Group middleware:**
```go
adminGroup := huma.NewGroup(api, "/admin")
adminGroup.UseMiddleware(AdminMiddleware)

huma.Get(adminGroup, "/users", listUsersHandler)
huma.Delete(adminGroup, "/users/{id}", deleteUserHandler)
```

See [references/middleware.md](references/middleware.md) for common patterns (auth, CORS, rate limiting, recovery).

### Testing

```go
func TestGetUser(t *testing.T) {
    _, api := humatest.New(t)
    huma.Get(api, "/users/{id}", getUserHandler)

    // Make request
    resp := api.Get("/users/123")

    // Assert
    assert.Equal(t, 200, resp.Code)
    assert.Contains(t, resp.Body.String(), "john")
}

// With body
func TestCreateUser(t *testing.T) {
    _, api := humatest.New(t)
    huma.Post(api, "/users", createUserHandler)

    resp := api.Post("/users", map[string]any{
        "username": "john",
        "email":    "john@example.com",
    })

    assert.Equal(t, 201, resp.Code)
}

// With headers
func TestAuth(t *testing.T) {
    _, api := humatest.New(t)
    huma.Get(api, "/protected", protectedHandler)

    resp := api.Get("/protected", "Authorization: Bearer token123")
    assert.Equal(t, 200, resp.Code)
}
```

See [references/testing.md](references/testing.md) for comprehensive testing patterns.

## Common Workflows

### CRUD Operations

```go
// List
type ListInput struct {
    Limit  int `query:"limit" default:"10" maximum:"100"`
    Offset int `query:"offset" default:"0" minimum:"0"`
}

type ListOutput struct {
    Body struct {
        Items []User `json:"items"`
        Total int    `json:"total"`
    }
}

huma.Get(api, "/users", listHandler)

// Create
type CreateInput struct {
    Body User
}

type CreateOutput struct {
    Status int `header:"-"`
    Body   struct {
        ID string `json:"id"`
    }
}

huma.Post(api, "/users", createHandler)

// Read
huma.Get(api, "/users/{id}", getHandler)

// Update
huma.Put(api, "/users/{id}", updateHandler)

// Delete
huma.Delete(api, "/users/{id}", deleteHandler)
```

### Authentication

Document security schemes:
```go
config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
    "bearer": {
        Type:         "http",
        Scheme:       "bearer",
        BearerFormat: "JWT",
    },
}
```

Apply to operations:
```go
huma.Register(api, huma.Operation{
    OperationID: "get-protected",
    Method:      http.MethodGet,
    Path:        "/protected",
    Security: []map[string][]string{
        {"bearer": {}},
    },
}, handler)
```

Implement middleware:
```go
func AuthMiddleware(api huma.API) func(huma.Context, func(huma.Context)) {
    return func(ctx huma.Context, next func(huma.Context)) {
        token := strings.TrimPrefix(ctx.Header("Authorization"), "Bearer ")

        if token == "" {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Missing token")
            return
        }

        // Validate token
        userID, err := validateToken(token)
        if err != nil {
            huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid token")
            return
        }

        ctx = huma.WithValue(ctx, "user_id", userID)
        next(ctx)
    }
}

api.UseMiddleware(AuthMiddleware(api))
```

See [references/auth.md](references/auth.md) for JWT, OAuth2, API keys, RBAC.

### Streaming & Server-Sent Events

**Basic streaming:**
```go
func streamHandler(ctx context.Context, input *struct{}) (*huma.StreamResponse, error) {
    return &huma.StreamResponse{
        Body: func(humaCtx huma.Context) {
            writer := humaCtx.BodyWriter()
            for i := 0; i < 10; i++ {
                fmt.Fprintf(writer, "Chunk %d\n", i)
                if f, ok := writer.(http.Flusher); ok {
                    f.Flush()
                }
                time.Sleep(100 * time.Millisecond)
            }
        },
    }, nil
}
```

**Server-Sent Events:**
```go
import "github.com/danielgtaylor/huma/v2/sse"

type ProgressEvent struct {
    Progress int    `json:"progress"`
    Message  string `json:"message"`
}

func sseHandler(ctx context.Context, input *struct{}, send sse.Sender) {
    for i := 0; i <= 100; i += 10 {
        send.Data(ProgressEvent{Progress: i, Message: fmt.Sprintf("Processing... %d%%", i)})
        time.Sleep(100 * time.Millisecond)
    }
}

sse.Register(api, huma.Operation{
    OperationID: "progress",
    Method:      http.MethodGet,
    Path:        "/progress",
}, map[string]any{"progress": ProgressEvent{}}, sseHandler)
```

See [references/streaming.md](references/streaming.md) for chat completion, logs, file downloads.

## Advanced Features

### Custom Validation (Resolvers)

```go
type Input struct {
    Username string `json:"username"`
    Email    string `json:"email"`
}

func (i *Input) Resolve(ctx huma.Context) []error {
    var errs []error

    // Normalize
    i.Username = strings.ToLower(strings.TrimSpace(i.Username))

    // Custom validation
    if strings.Contains(i.Username, " ") {
        errs = append(errs, &huma.ErrorDetail{
            Message:  "Username cannot contain spaces",
            Location: "body.username",
            Value:    i.Username,
        })
    }

    return errs
}
```

### File Upload (Multipart)

```go
type UploadInput struct {
    RawBody huma.MultipartFormFiles[struct {
        File  huma.FormFile   `form:"file" contentType:"image/*" required:"true"`
        Files []huma.FormFile `form:"files[]" contentType:"application/pdf"`
        Name  string          `form:"name" minLength:"1"`
    }]
}

func uploadHandler(ctx context.Context, input *UploadInput) (*struct{}, error) {
    data := input.RawBody.Data()

    // Read file
    content, err := io.ReadAll(data.File)
    if err != nil {
        return nil, huma.Error400BadRequest("Failed to read file")
    }

    // Process files
    for _, file := range data.Files {
        // Process each file
    }

    return &struct{}{}, nil
}
```

See [references/advanced.md](references/advanced.md) for transformers, auto-patch, conditional requests, custom formats.

## Project Setup

### Initialize with CLI

```go
type Options struct {
    Port int    `help:"Port to listen on" short:"p" default:"8080"`
    DB   string `help:"Database URL" default:"postgres://localhost/mydb"`
}

func main() {
    cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
        router := http.NewServeMux()
        api := humago.New(router, huma.DefaultConfig("My API", "1.0.0"))

        // Register operations
        huma.Get(api, "/health", healthHandler)

        server := &http.Server{
            Addr:    fmt.Sprintf(":%d", opts.Port),
            Handler: router,
        }

        hooks.OnStart(func() {
            fmt.Printf("Starting server on :%d\n", opts.Port)
            server.ListenAndServe()
        })

        hooks.OnStop(func() {
            server.Shutdown(context.Background())
        })
    })

    cli.Run()
}
```

Run with:
```bash
./app --port=8000
SERVICE_PORT=8000 ./app
```

### API Configuration

```go
config := huma.DefaultConfig("My API", "1.0.0")
config.Info.Description = "API description"
config.Info.Contact = &huma.Contact{
    Name:  "API Support",
    Email: "support@example.com",
}
config.Servers = []*huma.Server{
    {URL: "https://api.example.com", Description: "Production"},
    {URL: "https://api-staging.example.com", Description: "Staging"},
}

api := humago.New(router, config)
```

Access OpenAPI at `/openapi.json`, `/openapi.yaml`, and docs at `/docs`.

## Troubleshooting

**Validation not working:**
- Check field tags (minimum, maximum, pattern, etc.)
- Verify required/optional logic (omitempty, omitzero)
- Add custom validation with Resolvers

**Middleware not executing:**
- Check order: router middleware → API middleware → group middleware → operation middleware
- Verify next(ctx) is called
- Ensure operation has Security field if using auth middleware

**OpenAPI not showing endpoint:**
- Verify operation is registered with huma.Register or huma.Get/Post/etc.
- Check OperationID is unique
- Ensure handler signature matches HUMA pattern

**Streaming not working:**
- Flush after each write: `writer.(http.Flusher).Flush()`
- Set appropriate headers: Content-Type, Cache-Control
- Check for timeouts with long-running streams

## Resources

### references/
Comprehensive documentation for deep dives:
- **validation.md** - All validation tags, formats, patterns
- **errors.md** - Error handling, RFC 9457, patterns
- **middleware.md** - Common middleware (auth, CORS, logging, rate limit, recovery)
- **testing.md** - Testing patterns, mocks, integration tests
- **auth.md** - JWT, OAuth2, API keys, RBAC, multi-tenant
- **streaming.md** - Streaming, SSE, logs, file downloads, chat
- **advanced.md** - Resolvers, transformers, auto-patch, custom formats
- **architecture.md** - Project structures (flat, monolith, clean, microservices, hexagonal)

### assets/
Starter templates:
- **simple/** - Single-file API for prototypes
- **clean/** - Clean architecture for production apps
