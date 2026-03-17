# HUMA Advanced Features

## Custom Resolvers

Resolvers enable custom validation and data transformation:

```go
type CreateUserInput struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func (i *CreateUserInput) Resolve(ctx huma.Context) []error {
    var errs []error

    // Normalize data
    i.Username = strings.ToLower(strings.TrimSpace(i.Username))
    i.Email = strings.ToLower(strings.TrimSpace(i.Email))

    // Custom validation
    if strings.Contains(i.Username, " ") {
        errs = append(errs, &huma.ErrorDetail{
            Message:  "Username cannot contain spaces",
            Location: "body.username",
            Value:    i.Username,
        })
    }

    if len(i.Password) < 8 {
        errs = append(errs, &huma.ErrorDetail{
            Message:  "Password must be at least 8 characters",
            Location: "body.password",
            Value:    len(i.Password),
        })
    }

    return errs
}
```

## Resolvers with Path Buffer

For nested structures, use `PathBuffer`:

```go
type Address struct {
    Street string `json:"street"`
    City   string `json:"city"`
    Zip    string `json:"zip"`
}

func (a *Address) Resolve(ctx huma.Context, prefix *huma.PathBuffer) []error {
    var errs []error

    // Validate ZIP code format
    if matched, _ := regexp.MatchString(`^\d{5}$`, a.Zip); !matched {
        errs = append(errs, &huma.ErrorDetail{
            Message:  "ZIP code must be 5 digits",
            Location: prefix.With("zip"),
            Value:    a.Zip,
        })
    }

    return errs
}
```

## Response Transformers

Transform response data before serialization:

```go
func FieldFilterTransformer(ctx huma.Context, status string, v any) (any, error) {
    // Get requested fields from query
    fields := ctx.URL().Query().Get("fields")
    if fields == "" {
        return v, nil
    }

    // Filter response to only include requested fields
    requestedFields := strings.Split(fields, ",")

    // Use reflection to filter
    // ... filtering logic ...

    return filteredValue, nil
}

config.Transformers = []huma.Transformer{FieldFilterTransformer}
```

## Auto PATCH

Generate PATCH operations automatically:

```go
import "github.com/danielgtaylor/huma/v2/autopatch"

// Register GET and PUT first
huma.Get(api, "/users/{id}", getUserHandler)
huma.Put(api, "/users/{id}", updateUserHandler)

// Generate PATCH operations
autopatch.AutoPatch(api)
```

Supports:
- JSON Merge Patch (RFC 7386)
- JSON Patch (RFC 6902)
- Shorthand Merge Patch

Uses ETag/If-Match for conditional updates.

## Conditional Requests

### ETag Support

```go
type GetUserOutput struct {
    ETag string `header:"ETag"`
    Body User
}

func getUserHandler(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
    user, err := db.GetUser(input.ID)
    if err != nil {
        return nil, huma.Error404NotFound("User not found")
    }

    // Generate ETag from user data
    etag := fmt.Sprintf(`"%x"`, md5.Sum([]byte(user.UpdatedAt.String())))

    return &GetUserOutput{
        ETag: etag,
        Body: user,
    }, nil
}

type UpdateUserInput struct {
    ID      string `path:"id"`
    IfMatch string `header:"If-Match"`
    Body    User
}

func updateUserHandler(ctx context.Context, input *UpdateUserInput) (*UpdateUserOutput, error) {
    user, err := db.GetUser(input.ID)
    if err != nil {
        return nil, huma.Error404NotFound("User not found")
    }

    currentETag := fmt.Sprintf(`"%x"`, md5.Sum([]byte(user.UpdatedAt.String())))

    if input.IfMatch != "" && input.IfMatch != currentETag {
        return nil, huma.Error412PreconditionFailed("Resource has been modified")
    }

    // Update user
    // ...

    return &UpdateUserOutput{}, nil
}
```

### Last-Modified Support

```go
type GetResourceOutput struct {
    LastModified time.Time `header:"Last-Modified" timeFormat:"Mon, 02 Jan 2006 15:04:05 GMT"`
    Body         Resource
}

type UpdateResourceInput struct {
    IfUnmodifiedSince *time.Time `header:"If-Unmodified-Since" timeFormat:"Mon, 02 Jan 2006 15:04:05 GMT"`
    Body              Resource
}
```

## Content Negotiation

### Custom Formats

```go
type XMLFormat struct{}

func (f XMLFormat) Marshal(w io.Writer, v any) error {
    return xml.NewEncoder(w).Encode(v)
}

func (f XMLFormat) Unmarshal(r io.Reader, v any) error {
    return xml.NewDecoder(r).Decode(v)
}

config.Formats = map[string]huma.Format{
    "application/json": huma.DefaultJSONFormat,
    "application/xml":  XMLFormat{},
}
```

### CBOR Support

```go
import _ "github.com/danielgtaylor/huma/v2/formats/cbor"

// CBOR is automatically registered as application/cbor
```

## Groups

Group operations with shared configuration:

```go
// Create group with path prefixes
v1 := huma.NewGroup(api, "/v1")
v2 := huma.NewGroup(api, "/v2")

// Apply middleware to group
v1.UseMiddleware(legacyAuthMiddleware)
v2.UseMiddleware(modernAuthMiddleware)

// Apply transformers
v1.UseTransformer(v1Transformer)

// Apply modifiers
v2.UseModifier(func(op *huma.Operation, next func(*huma.Operation)) {
    op.Tags = []string{"v2"}
    next(op)
})

// Register operations
huma.Get(v1, "/users", listUsersV1Handler)
huma.Get(v2, "/users", listUsersV2Handler)
```

## Multiple Path Prefixes

```go
// Register at multiple paths
grp := huma.NewGroup(api, "/v1", "/v2")

huma.Get(grp, "/users", handler)
// Accessible at both /v1/users and /v2/users
```

## Operation Modifiers

```go
func TagModifier(tag string) func(*huma.Operation, func(*huma.Operation)) {
    return func(op *huma.Operation, next func(*huma.Operation)) {
        op.Tags = append(op.Tags, tag)
        next(op)
    }
}

adminGroup := huma.NewGroup(api, "/admin")
adminGroup.UseModifier(TagModifier("admin"))
```

## Deprecating Operations

```go
huma.Register(api, huma.Operation{
    OperationID: "old-endpoint",
    Method:      http.MethodGet,
    Path:        "/old",
    Deprecated:  true,
    Summary:     "Old endpoint (deprecated)",
}, handler)
```

## External Documentation

```go
huma.Register(api, huma.Operation{
    OperationID: "complex-operation",
    Method:      http.MethodPost,
    Path:        "/complex",
    ExternalDocs: &huma.ExternalDocs{
        Description: "Full documentation",
        URL:         "https://docs.example.com/complex-operation",
    },
}, handler)
```

## Request/Response Examples

```go
type CreateUserInput struct {
    Body struct {
        Username string `json:"username" example:"john_doe"`
        Email    string `json:"email" example:"john@example.com"`
        Age      int    `json:"age" example:"25"`
    }
}

huma.Register(api, huma.Operation{
    OperationID: "create-user",
    Method:      http.MethodPost,
    Path:        "/users",
    Examples: map[string]*huma.Example{
        "basic": {
            Summary: "Basic user",
            Value: map[string]any{
                "username": "john_doe",
                "email":    "john@example.com",
                "age":      25,
            },
        },
        "admin": {
            Summary: "Admin user",
            Value: map[string]any{
                "username": "admin",
                "email":    "admin@example.com",
                "age":      30,
                "role":     "admin",
            },
        },
    },
}, handler)
```

## Custom Error Types

```go
type CustomError struct {
    status  int
    msg     string
    Code    string   `json:"code"`
    Details []string `json:"details,omitempty"`
}

func (e CustomError) Error() string       { return e.msg }
func (e CustomError) GetStatus() int      { return e.status }

func NewCustomError(status int, msg, code string, details ...string) CustomError {
    return CustomError{
        status:  status,
        msg:     msg,
        Code:    code,
        Details: details,
    }
}

// Override global error creator
huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
    details := make([]string, len(errs))
    for i, err := range errs {
        details[i] = err.Error()
    }
    return NewCustomError(status, msg, "CUSTOM_ERROR", details...)
}
```

## Schema Customization

```go
// Custom schema for a type
huma.Register(api, huma.Operation{
    OperationID: "get-thing",
    Method:      http.MethodGet,
    Path:        "/thing",
    Responses: map[string]*huma.Response{
        "200": {
            Description: "Success",
            Content: map[string]*huma.MediaType{
                "application/json": {
                    Schema: &huma.Schema{
                        Type: "object",
                        Properties: map[string]*huma.Schema{
                            "custom": {Type: "string"},
                        },
                    },
                },
            },
        },
    },
}, handler)
```

## Multipart Form Files

```go
type UploadInput struct {
    RawBody huma.MultipartFormFiles[struct {
        Photo huma.FormFile   `form:"photo" contentType:"image/*" required:"true"`
        Docs  []huma.FormFile `form:"docs[]" contentType:"application/pdf"`
        Name  string          `form:"name"`
        Tags  []string        `form:"tags"`
    }]
}

func uploadHandler(ctx context.Context, input *UploadInput) (*struct{}, error) {
    data := input.RawBody.Data()

    // Read photo
    photoContent, err := io.ReadAll(data.Photo)
    if err != nil {
        return nil, huma.Error400BadRequest("Failed to read photo")
    }

    // Process documents
    for _, doc := range data.Docs {
        content, _ := io.ReadAll(doc)
        // Process document
    }

    // Access form fields
    name := data.Name
    tags := data.Tags

    return &struct{}{}, nil
}
```

## Dynamic Responses

```go
func handler(ctx context.Context, input *MyInput) (*MyOutput, error) {
    output := &MyOutput{}

    if input.IncludeExtra {
        output.Status = 200
        output.Body = DetailedResponse{}
    } else {
        output.Status = 200
        output.Body = SimpleResponse{}
    }

    return output, nil
}
```

## Best Practices

1. **Use resolvers for complex validation** - Keep tag-based validation simple
2. **Document transformers** - Make behavior clear to API consumers
3. **Use groups for versioning** - Keep v1 and v2 separate
4. **Leverage auto-patch** - Reduce boilerplate for update operations
5. **Use ETags for concurrency** - Prevent lost updates
6. **Provide examples** - Help API consumers understand usage
7. **Custom formats sparingly** - JSON/CBOR cover most needs
8. **Deprecate properly** - Mark old endpoints, provide alternatives
