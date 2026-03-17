# HUMA Testing Patterns

## Test Utilities

HUMA provides `humatest` package for easy testing:

```go
import (
    "testing"
    "github.com/danielgtaylor/huma/v2/humatest"
)

func TestMyAPI(t *testing.T) {
    _, api := humatest.New(t)

    // Register operations
    huma.Get(api, "/greeting/{name}", greetingHandler)

    // Make requests
    resp := api.Get("/greeting/world")

    // Assert response
    if resp.Code != 200 {
        t.Errorf("Expected 200, got %d", resp.Code)
    }
}
```

## Basic Handler Testing

```go
func greetingHandler(ctx context.Context, input *struct {
    Name string `path:"name"`
}) (*struct {
    Body struct {
        Message string `json:"message"`
    }
}, error) {
    resp := &struct {
        Body struct {
            Message string `json:"message"`
        }
    }{}
    resp.Body.Message = fmt.Sprintf("Hello, %s!", input.Name)
    return resp, nil
}

func TestGreeting(t *testing.T) {
    _, api := humatest.New(t)
    huma.Get(api, "/greeting/{name}", greetingHandler)

    resp := api.Get("/greeting/world")

    assert.Equal(t, 200, resp.Code)
    assert.Contains(t, resp.Body.String(), "Hello, world!")
}
```

## Testing with Request Body

```go
func TestCreateUser(t *testing.T) {
    _, api := humatest.New(t)
    huma.Post(api, "/users", createUserHandler)

    // POST with map
    resp := api.Post("/users", map[string]any{
        "username": "john",
        "email":    "john@example.com",
    })

    assert.Equal(t, 201, resp.Code)

    // Or POST with struct
    input := CreateUserInput{
        Username: "jane",
        Email:    "jane@example.com",
    }
    resp = api.Post("/users", input)

    assert.Equal(t, 201, resp.Code)
}
```

## Testing with Headers

```go
func TestAuth(t *testing.T) {
    _, api := humatest.New(t)
    huma.Get(api, "/protected", protectedHandler)

    // Without auth header
    resp := api.Get("/protected")
    assert.Equal(t, 401, resp.Code)

    // With auth header
    resp = api.Get("/protected", "Authorization: Bearer token123")
    assert.Equal(t, 200, resp.Code)
}
```

## Testing with Query Parameters

```go
func TestList(t *testing.T) {
    _, api := humatest.New(t)
    huma.Get(api, "/items", listHandler)

    resp := api.Get("/items?limit=10&offset=20")

    assert.Equal(t, 200, resp.Code)
}
```

## Testing Validation Errors

```go
func TestValidation(t *testing.T) {
    _, api := humatest.New(t)
    huma.Post(api, "/users", createUserHandler)

    // Invalid input - missing required field
    resp := api.Post("/users", map[string]any{
        "username": "john",
        // email is missing
    })

    assert.Equal(t, 422, resp.Code)
    assert.Contains(t, resp.Body.String(), "email")

    // Invalid input - validation failure
    resp = api.Post("/users", map[string]any{
        "username": "ab", // too short
        "email":    "invalid-email",
    })

    assert.Equal(t, 422, resp.Code)
}
```

## Testing Error Responses

```go
func TestNotFound(t *testing.T) {
    _, api := humatest.New(t)
    huma.Get(api, "/users/{id}", getUserHandler)

    resp := api.Get("/users/nonexistent")

    assert.Equal(t, 404, resp.Code)

    var errorResp huma.ErrorModel
    json.Unmarshal(resp.Body.Bytes(), &errorResp)

    assert.Equal(t, 404, errorResp.Status)
    assert.Contains(t, errorResp.Detail, "not found")
}
```

## Testing with Middleware

```go
func TestWithMiddleware(t *testing.T) {
    _, api := humatest.New(t)

    // Add middleware
    api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
        ctx.SetHeader("X-Custom", "value")
        next(ctx)
    })

    huma.Get(api, "/test", testHandler)

    resp := api.Get("/test")

    assert.Equal(t, "value", resp.Header().Get("X-Custom"))
}
```

## Testing Status Codes

```go
func TestStatusCodes(t *testing.T) {
    _, api := humatest.New(t)

    huma.Post(api, "/items", createHandler)
    huma.Get(api, "/items/{id}", getHandler)
    huma.Put(api, "/items/{id}", updateHandler)
    huma.Delete(api, "/items/{id}", deleteHandler)

    // 201 Created
    resp := api.Post("/items", map[string]any{"name": "test"})
    assert.Equal(t, 201, resp.Code)

    // 200 OK
    resp = api.Get("/items/123")
    assert.Equal(t, 200, resp.Code)

    // 200 OK
    resp = api.Put("/items/123", map[string]any{"name": "updated"})
    assert.Equal(t, 200, resp.Code)

    // 204 No Content
    resp = api.Delete("/items/123")
    assert.Equal(t, 204, resp.Code)
}
```

## Table-Driven Tests

```go
func TestGreetingVariations(t *testing.T) {
    _, api := humatest.New(t)
    huma.Get(api, "/greeting/{name}", greetingHandler)

    tests := []struct {
        name     string
        path     string
        wantCode int
        wantBody string
    }{
        {
            name:     "valid name",
            path:     "/greeting/world",
            wantCode: 200,
            wantBody: "Hello, world!",
        },
        {
            name:     "long name",
            path:     "/greeting/" + strings.Repeat("a", 31),
            wantCode: 422, // exceeds maxLength
            wantBody: "validation",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp := api.Get(tt.path)
            assert.Equal(t, tt.wantCode, resp.Code)
            assert.Contains(t, resp.Body.String(), tt.wantBody)
        })
    }
}
```

## Testing JSON Response Structure

```go
func TestResponseStructure(t *testing.T) {
    _, api := humatest.New(t)
    huma.Get(api, "/users/{id}", getUserHandler)

    resp := api.Get("/users/123")

    assert.Equal(t, 200, resp.Code)

    var result struct {
        ID       string `json:"id"`
        Username string `json:"username"`
        Email    string `json:"email"`
    }

    err := json.Unmarshal(resp.Body.Bytes(), &result)
    assert.NoError(t, err)

    assert.Equal(t, "123", result.ID)
    assert.NotEmpty(t, result.Username)
    assert.NotEmpty(t, result.Email)
}
```

## Testing with External Dependencies

```go
type UserService interface {
    GetUser(id string) (*User, error)
}

type mockUserService struct {
    users map[string]*User
}

func (m *mockUserService) GetUser(id string) (*User, error) {
    user, ok := m.users[id]
    if !ok {
        return nil, fmt.Errorf("user not found")
    }
    return user, nil
}

func TestWithMock(t *testing.T) {
    _, api := humatest.New(t)

    mockService := &mockUserService{
        users: map[string]*User{
            "123": {ID: "123", Username: "john"},
        },
    }

    handler := makeGetUserHandler(mockService)
    huma.Get(api, "/users/{id}", handler)

    // Test success case
    resp := api.Get("/users/123")
    assert.Equal(t, 200, resp.Code)

    // Test not found case
    resp = api.Get("/users/999")
    assert.Equal(t, 404, resp.Code)
}
```

## Testing Custom Resolvers

```go
type InputWithResolver struct {
    Value int `json:"value"`
}

func (i *InputWithResolver) Resolve(ctx huma.Context) []error {
    if i.Value < 0 {
        return []error{&huma.ErrorDetail{
            Message:  "Value must be positive",
            Location: "body.value",
            Value:    i.Value,
        }}
    }
    return nil
}

func TestResolver(t *testing.T) {
    _, api := humatest.New(t)
    huma.Post(api, "/items", createHandler)

    // Valid input
    resp := api.Post("/items", map[string]any{"value": 10})
    assert.Equal(t, 201, resp.Code)

    // Invalid input - resolver catches it
    resp = api.Post("/items", map[string]any{"value": -5})
    assert.Equal(t, 422, resp.Code)
    assert.Contains(t, resp.Body.String(), "must be positive")
}
```

## Integration Testing

```go
func TestFullWorkflow(t *testing.T) {
    _, api := humatest.New(t)

    // Register all operations
    huma.Post(api, "/users", createUserHandler)
    huma.Get(api, "/users/{id}", getUserHandler)
    huma.Put(api, "/users/{id}", updateUserHandler)
    huma.Delete(api, "/users/{id}", deleteUserHandler)

    // Create user
    createResp := api.Post("/users", map[string]any{
        "username": "john",
        "email":    "john@example.com",
    })
    assert.Equal(t, 201, createResp.Code)

    var created struct {
        ID string `json:"id"`
    }
    json.Unmarshal(createResp.Body.Bytes(), &created)

    // Get user
    getResp := api.Get("/users/" + created.ID)
    assert.Equal(t, 200, getResp.Code)

    // Update user
    updateResp := api.Put("/users/"+created.ID, map[string]any{
        "email": "newemail@example.com",
    })
    assert.Equal(t, 200, updateResp.Code)

    // Delete user
    deleteResp := api.Delete("/users/" + created.ID)
    assert.Equal(t, 204, deleteResp.Code)

    // Verify deleted
    getResp = api.Get("/users/" + created.ID)
    assert.Equal(t, 404, getResp.Code)
}
```

## Testing Best Practices

1. **Use table-driven tests** - Test multiple scenarios efficiently
2. **Test error cases** - Validation, not found, unauthorized, etc.
3. **Mock external dependencies** - Database, APIs, services
4. **Test the full request-response cycle** - Including middleware
5. **Verify response structure** - Not just status codes
6. **Use descriptive test names** - Make failures easy to understand
7. **Test edge cases** - Empty inputs, boundary values, special characters
8. **Keep tests independent** - Each test should be able to run alone
