# HUMA Error Handling

## Error Model (RFC 9457)

HUMA uses RFC 9457 Problem Details:

```go
type ErrorModel struct {
    Type     string         `json:"type,omitempty"`      // URI identifying error type
    Title    string         `json:"title,omitempty"`     // Short human-readable summary
    Status   int            `json:"status,omitempty"`    // HTTP status code
    Detail   string         `json:"detail,omitempty"`    // Detailed explanation
    Instance string         `json:"instance,omitempty"`  // URI identifying this occurrence
    Errors   []*ErrorDetail `json:"errors,omitempty"`    // Detailed error list
}

type ErrorDetail struct {
    Message  string `json:"message,omitempty"`  // Error message
    Location string `json:"location,omitempty"` // e.g., "body.email", "query.limit"
    Value    any    `json:"value,omitempty"`    // The invalid value
}
```

## Built-in Error Functions

```go
// 4xx Client Errors
huma.Error400BadRequest(msg, ...errs)
huma.Error401Unauthorized(msg, ...errs)
huma.Error403Forbidden(msg, ...errs)
huma.Error404NotFound(msg, ...errs)
huma.Error405MethodNotAllowed(msg, ...errs)
huma.Error406NotAcceptable(msg, ...errs)
huma.Error409Conflict(msg, ...errs)
huma.Error410Gone(msg, ...errs)
huma.Error412PreconditionFailed(msg, ...errs)
huma.Error413RequestEntityTooLarge(msg, ...errs)
huma.Error415UnsupportedMediaType(msg, ...errs)
huma.Error422UnprocessableEntity(msg, ...errs)
huma.Error429TooManyRequests(msg, ...errs)

// 5xx Server Errors
huma.Error500InternalServerError(msg, ...errs)
huma.Error501NotImplemented(msg, ...errs)
huma.Error502BadGateway(msg, ...errs)
huma.Error503ServiceUnavailable(msg, ...errs)
huma.Error504GatewayTimeout(msg, ...errs)
```

## Basic Error Usage

```go
func handler(ctx context.Context, input *MyInput) (*MyOutput, error) {
    if input.ID == "" {
        return nil, huma.Error400BadRequest("ID is required")
    }

    item, err := db.Get(input.ID)
    if err == sql.ErrNoRows {
        return nil, huma.Error404NotFound("Item not found")
    }
    if err != nil {
        return nil, huma.Error500InternalServerError("Database error", err)
    }

    return &MyOutput{Body: item}, nil
}
```

## Detailed Errors

```go
func handler(ctx context.Context, input *MyInput) (*MyOutput, error) {
    if input.Email == "" {
        return nil, huma.Error422UnprocessableEntity(
            "Validation failed",
            &huma.ErrorDetail{
                Message:  "Email is required",
                Location: "body.email",
                Value:    "",
            },
        )
    }

    if !isValidEmail(input.Email) {
        return nil, huma.Error422UnprocessableEntity(
            "Invalid email format",
            &huma.ErrorDetail{
                Message:  "Must be a valid email address",
                Location: "body.email",
                Value:    input.Email,
            },
        )
    }

    return &MyOutput{}, nil
}
```

## Multiple Errors

```go
func handler(ctx context.Context, input *CreateUserInput) (*CreateUserOutput, error) {
    var errs []*huma.ErrorDetail

    if input.Username == "" {
        errs = append(errs, &huma.ErrorDetail{
            Message:  "Username is required",
            Location: "body.username",
            Value:    "",
        })
    }

    if len(input.Password) < 8 {
        errs = append(errs, &huma.ErrorDetail{
            Message:  "Password must be at least 8 characters",
            Location: "body.password",
            Value:    len(input.Password),
        })
    }

    if len(errs) > 0 {
        return nil, huma.Error422UnprocessableEntity("Validation failed", errs...)
    }

    return &CreateUserOutput{}, nil
}
```

## Custom Error Types

Define custom error struct implementing `huma.StatusError`:

```go
type MyError struct {
    status int
    msg    string
    Code   string `json:"code"`
}

func (e MyError) Error() string       { return e.msg }
func (e MyError) GetStatus() int      { return e.status }

func NewMyError(status int, msg, code string) MyError {
    return MyError{status: status, msg: msg, Code: code}
}

// Usage
return nil, NewMyError(400, "Invalid request", "INVALID_INPUT")
```

Override `huma.NewError` to customize error creation:

```go
huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
    return NewMyError(status, msg, "CUSTOM_CODE")
}
```

## Error Priority

When multiple error conditions exist:

1. **Panic** → 500 Internal Server Error
2. **Body too large** → 413 Request Entity Too Large
3. **Timeout** → 408 Request Timeout
4. **Parse failure** → 400 Bad Request
5. **Validation failure** → 422 Unprocessable Entity
6. **Handler error** → Status from error or 500

## Error Handling Patterns

### Business Logic Errors

```go
func handler(ctx context.Context, input *TransferInput) (*TransferOutput, error) {
    balance, err := getBalance(input.AccountID)
    if err != nil {
        return nil, huma.Error500InternalServerError("Failed to get balance")
    }

    if balance < input.Amount {
        return nil, huma.Error409Conflict("Insufficient funds")
    }

    return &TransferOutput{}, nil
}
```

### Database Errors

```go
func handler(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
    user, err := db.GetUser(input.ID)
    if err == sql.ErrNoRows {
        return nil, huma.Error404NotFound("User not found")
    }
    if err != nil {
        // Log the actual error for debugging
        log.Printf("Database error: %v", err)
        // Return generic error to client
        return nil, huma.Error500InternalServerError("Failed to retrieve user")
    }

    return &GetUserOutput{Body: user}, nil
}
```

### Authorization Errors

```go
func handler(ctx context.Context, input *DeleteInput) (*DeleteOutput, error) {
    userID := ctx.Context().Value("user_id").(string)

    resource, err := db.Get(input.ID)
    if err != nil {
        return nil, huma.Error404NotFound("Resource not found")
    }

    if resource.OwnerID != userID {
        return nil, huma.Error403Forbidden("You don't have permission to delete this resource")
    }

    return &DeleteOutput{}, nil
}
```

### Conflict Errors

```go
func handler(ctx context.Context, input *CreateInput) (*CreateOutput, error) {
    exists, err := db.Exists(input.Username)
    if err != nil {
        return nil, huma.Error500InternalServerError("Failed to check username")
    }

    if exists {
        return nil, huma.Error409Conflict("Username already taken")
    }

    return &CreateOutput{}, nil
}
```

## Error Response Examples

**404 Not Found:**
```json
{
  "title": "Not Found",
  "status": 404,
  "detail": "Item not found"
}
```

**422 Validation Error:**
```json
{
  "title": "Unprocessable Entity",
  "status": 422,
  "detail": "Validation failed",
  "errors": [
    {
      "message": "Must be at least 8 characters",
      "location": "body.password",
      "value": 5
    },
    {
      "message": "Email is required",
      "location": "body.email",
      "value": ""
    }
  ]
}
```

## Best Practices

1. **Return all errors at once** - Don't fail on first error, collect all validation errors
2. **Be specific** - Use appropriate status codes (422 for validation, 409 for conflicts, etc.)
3. **Don't leak internals** - Log detailed errors server-side, return generic messages to client
4. **Include location** - Always specify which field caused the error
5. **Provide helpful messages** - Error messages should guide users to fix the issue
6. **Use consistent error format** - RFC 9457 provides standard structure
