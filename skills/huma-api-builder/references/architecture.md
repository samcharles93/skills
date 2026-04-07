# HUMA Project Architecture Patterns

## Flat/Simple Structure

Best for: Small APIs, prototypes, learning

```
project/
├── main.go           # Everything in one file or a few files
├── go.mod
└── go.sum
```

**main.go example:**
```go
package main

import (
    "context"
    "fmt"
    "net/http"

    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/adapters/humago"
    "github.com/danielgtaylor/huma/v2/humacli"
)

type GreetingInput struct {
    Name string `path:"name" maxLength:"30"`
}

type GreetingOutput struct {
    Body struct {
        Message string `json:"message"`
    }
}

func greetingHandler(ctx context.Context, input *GreetingInput) (*GreetingOutput, error) {
    resp := &GreetingOutput{}
    resp.Body.Message = fmt.Sprintf("Hello, %s!", input.Name)
    return resp, nil
}

type Options struct {
    Port int `help:"Port to listen on" default:"8080"`
}

func main() {
    cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
        router := http.NewServeMux()
        api := humago.New(router, huma.DefaultConfig("My API", "1.0.0"))

        huma.Get(api, "/greeting/{name}", greetingHandler)

        server := &http.Server{
            Addr:    fmt.Sprintf(":%d", opts.Port),
            Handler: router,
        }

        hooks.OnStart(func() {
            fmt.Printf("Starting server on :%d...\n", opts.Port)
            server.ListenAndServe()
        })

        hooks.OnStop(func() {
            server.Shutdown(context.Background())
        })
    })

    cli.Run()
}
```

## Monolith Structure

Best for: Small to medium APIs, single deployable

```
project/
├── cmd/
│   └── api/
│       └── main.go       # CLI setup, server startup
├── handlers.go           # All handlers
├── models.go             # Input/output structs
├── middleware.go         # Middleware functions
├── database.go           # Database connection
├── config.go             # Configuration
├── go.mod
└── go.sum
```

**main.go:**
```go
package main

import (
    "fmt"
    "net/http"

    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/adapters/humago"
    "github.com/danielgtaylor/huma/v2/humacli"
)

type Options struct {
    Port int    `help:"Port to listen on" default:"8080"`
    DB   string `help:"Database connection string" default:"postgres://..."`
}

func main() {
    cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
        // Initialize database
        db := initDB(opts.DB)

        // Setup router and API
        router := http.NewServeMux()
        config := huma.DefaultConfig("My API", "1.0.0")
        api := humago.New(router, config)

        // Add middleware
        api.UseMiddleware(loggingMiddleware)
        api.UseMiddleware(authMiddleware(api, db))

        // Register routes
        registerRoutes(api, db)

        server := &http.Server{
            Addr:    fmt.Sprintf(":%d", opts.Port),
            Handler: router,
        }

        hooks.OnStart(func() {
            fmt.Printf("Starting server on :%d...\n", opts.Port)
            server.ListenAndServe()
        })

        hooks.OnStop(func() {
            server.Shutdown(context.Background())
            db.Close()
        })
    })

    cli.Run()
}
```

**handlers.go:**
```go
package main

func registerRoutes(api huma.API, db *Database) {
    huma.Get(api, "/users", listUsersHandler(db))
    huma.Post(api, "/users", createUserHandler(db))
    huma.Get(api, "/users/{id}", getUserHandler(db))
    huma.Put(api, "/users/{id}", updateUserHandler(db))
    huma.Delete(api, "/users/{id}", deleteUserHandler(db))
}

func getUserHandler(db *Database) func(context.Context, *GetUserInput) (*GetUserOutput, error) {
    return func(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
        user, err := db.GetUser(input.ID)
        if err != nil {
            return nil, huma.Error404NotFound("User not found")
        }

        return &GetUserOutput{Body: user}, nil
    }
}
```

## Clean Architecture

Best for: Medium to large APIs, clear separation of concerns

```
project/
├── cmd/
│   └── api/
│       └── main.go                 # Entry point
├── internal/
│   ├── api/
│   │   ├── api.go                  # API setup
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   ├── logging.go
│   │   │   └── cors.go
│   │   ├── handlers/
│   │   │   ├── users.go
│   │   │   ├── products.go
│   │   │   └── orders.go
│   │   └── models/
│   │       ├── user.go
│   │       ├── product.go
│   │       └── order.go
│   ├── domain/
│   │   ├── user/
│   │   │   ├── user.go             # Domain entities
│   │   │   ├── service.go          # Business logic
│   │   │   └── repository.go       # Interface
│   │   ├── product/
│   │   │   └── ...
│   │   └── order/
│   │       └── ...
│   ├── repository/
│   │   ├── postgres/
│   │   │   ├── user.go
│   │   │   ├── product.go
│   │   │   └── order.go
│   │   └── repository.go           # Common code
│   └── config/
│       └── config.go
├── pkg/                            # Reusable packages
│   └── jwt/
│       └── jwt.go
├── go.mod
└── go.sum
```

**cmd/api/main.go:**
```go
package main

import (
    "context"
    "fmt"
    "net/http"

    "myapp/internal/api"
    "myapp/internal/config"
    "myapp/internal/domain/user"
    "myapp/internal/repository/postgres"

    "github.com/danielgtaylor/huma/v2/humacli"
)

type Options struct {
    Port int    `help:"Port" default:"8080"`
    DB   string `help:"Database URL" default:"postgres://..."`
}

func main() {
    cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
        cfg := config.Load()

        // Setup repositories
        db := postgres.Connect(opts.DB)
        userRepo := postgres.NewUserRepository(db)

        // Setup services
        userService := user.NewService(userRepo)

        // Setup API
        humaAPI := api.New(cfg, api.Dependencies{
            UserService: userService,
        })

        server := &http.Server{
            Addr:    fmt.Sprintf(":%d", opts.Port),
            Handler: humaAPI.Router(),
        }

        hooks.OnStart(func() {
            server.ListenAndServe()
        })

        hooks.OnStop(func() {
            server.Shutdown(context.Background())
            db.Close()
        })
    })

    cli.Run()
}
```

**internal/api/api.go:**
```go
package api

import (
    "net/http"

    "myapp/internal/api/handlers"
    "myapp/internal/api/middleware"
    "myapp/internal/config"
    "myapp/internal/domain/user"

    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/adapters/humago"
)

type Dependencies struct {
    UserService user.Service
}

type API struct {
    router http.Handler
    huma   huma.API
}

func New(cfg *config.Config, deps Dependencies) *API {
    router := http.NewServeMux()

    config := huma.DefaultConfig("My API", "1.0.0")
    humaAPI := humago.New(router, config)

    // Middleware
    humaAPI.UseMiddleware(middleware.Logging())
    humaAPI.UseMiddleware(middleware.Auth(humaAPI, cfg.JWTSecret))

    // Handlers
    handlers.RegisterUserRoutes(humaAPI, deps.UserService)

    return &API{
        router: router,
        huma:   humaAPI,
    }
}

func (a *API) Router() http.Handler {
    return a.router
}
```

**internal/api/handlers/users.go:**
```go
package handlers

import (
    "context"

    "myapp/internal/api/models"
    "myapp/internal/domain/user"

    "github.com/danielgtaylor/huma/v2"
)

func RegisterUserRoutes(api huma.API, service user.Service) {
    huma.Get(api, "/users/{id}", GetUser(service))
    huma.Post(api, "/users", CreateUser(service))
}

func GetUser(service user.Service) func(context.Context, *models.GetUserInput) (*models.GetUserOutput, error) {
    return func(ctx context.Context, input *models.GetUserInput) (*models.GetUserOutput, error) {
        user, err := service.GetByID(ctx, input.ID)
        if err != nil {
            return nil, huma.Error404NotFound("User not found")
        }

        return &models.GetUserOutput{
            Body: models.UserFromDomain(user),
        }, nil
    }
}
```

## Microservices Structure

Best for: Distributed systems, independent services

```
services/
├── user-service/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── api/
│   │   ├── domain/
│   │   └── repository/
│   ├── go.mod
│   └── go.sum
├── product-service/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   └── ...
│   ├── go.mod
│   └── go.sum
├── order-service/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   └── ...
│   ├── go.mod
│   └── go.sum
└── shared/
    └── pkg/
        ├── auth/
        ├── httpclient/
        └── events/
```

Each service follows clean architecture internally, with shared packages for cross-cutting concerns.

## Monorepo Structure

Best for: Multiple related services, shared code

```
monorepo/
├── services/
│   ├── api/
│   │   ├── main.go
│   │   └── internal/...
│   ├── worker/
│   │   ├── main.go
│   │   └── internal/...
│   └── admin/
│       ├── main.go
│       └── internal/...
├── pkg/                     # Shared packages
│   ├── auth/
│   ├── database/
│   ├── events/
│   └── models/
├── tools/                   # Development tools
│   └── codegen/
├── go.work                  # Go workspace
├── go.work.sum
└── README.md
```

**go.work:**
```
go 1.22

use (
    ./services/api
    ./services/worker
    ./services/admin
    ./pkg/auth
    ./pkg/database
)
```

## Hexagonal/Ports & Adapters

Best for: Testability, adapter swapping

```
project/
├── cmd/api/main.go
├── internal/
│   ├── core/
│   │   ├── domain/
│   │   │   └── user.go         # Entities
│   │   ├── ports/
│   │   │   ├── input/          # Use cases (interfaces)
│   │   │   │   └── user.go
│   │   │   └── output/         # Repository interfaces
│   │   │       └── user.go
│   │   └── services/           # Business logic
│   │       └── user.go
│   └── adapters/
│       ├── input/
│       │   └── http/           # HTTP handlers
│       │       └── user.go
│       └── output/
│           ├── postgres/       # DB implementation
│           │   └── user.go
│           └── redis/          # Cache implementation
│               └── user.go
├── go.mod
└── go.sum
```

## Feature-Based Structure

Best for: Large APIs, team organization

```
project/
├── cmd/api/main.go
├── internal/
│   ├── users/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── models.go
│   │   └── user.go
│   ├── products/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── models.go
│   │   └── product.go
│   ├── orders/
│   │   └── ...
│   └── shared/
│       ├── database/
│       ├── middleware/
│       └── config/
├── go.mod
└── go.sum
```

## Choosing an Architecture

**Use Flat/Simple when:**
- Learning HUMA
- Building prototypes
- API has < 5 endpoints
- Single developer

**Use Monolith when:**
- Small to medium API
- Fast development needed
- Single deployment unit
- Team < 5 people

**Use Clean Architecture when:**
- Medium to large API
- Long-term maintenance
- Clear separation needed
- Team > 5 people

**Use Microservices when:**
- Multiple independent services
- Different scaling needs
- Separate deployment cycles
- Large organization

**Use Monorepo when:**
- Related services
- Shared code/packages
- Coordinated releases
- Single team managing multiple services

**Use Hexagonal when:**
- Testability is critical
- Need to swap adapters
- Complex business logic
- Independent of frameworks

**Use Feature-Based when:**
- Large API with many features
- Team organized by features
- Vertical slice development
- Reduce merge conflicts
