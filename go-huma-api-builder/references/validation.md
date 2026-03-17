# HUMA Validation Patterns

## Validation Tags

### Numeric Constraints
```go
type Product struct {
    Price    float64 `json:"price" minimum:"0" exclusiveMinimum:"true"`
    Quantity int     `json:"quantity" minimum:"1" maximum:"1000"`
    Discount float64 `json:"discount" minimum:"0" maximum:"100" multipleOf:"0.01"`
}
```

### String Constraints
```go
type User struct {
    Username string `json:"username" minLength:"3" maxLength:"20" pattern:"^[a-zA-Z0-9_]+$"`
    Email    string `json:"email" format:"email"`
    Website  string `json:"website" format:"uri"`
    Bio      string `json:"bio" maxLength:"500"`
}
```

### Array Constraints
```go
type Order struct {
    Items []string `json:"items" minItems:"1" maxItems:"100" uniqueItems:"true"`
    Tags  []string `json:"tags" maxItems:"10"`
}
```

### Object Constraints
```go
type Config struct {
    Settings map[string]string `json:"settings" minProperties:"1" maxProperties:"50"`
}
```

### Format Validators

**Date/Time:**
- `date-time` - RFC3339 date-time
- `date-time-http` - HTTP date format
- `date` - RFC3339 date (YYYY-MM-DD)
- `time` - RFC3339 time (HH:MM:SS)
- `duration` - ISO 8601 duration

**Network:**
- `email` - Email address
- `idn-email` - International email
- `hostname` - DNS hostname
- `ipv4` - IPv4 address
- `ipv6` - IPv6 address
- `uri` - URI
- `uri-reference` - URI reference

**Special:**
- `uuid` - UUID
- `json-pointer` - JSON pointer
- `uri-template` - URI template
- `regex` - Regular expression
- `base64` - Base64 encoded

```go
type Event struct {
    ID        string    `json:"id" format:"uuid"`
    Timestamp time.Time `json:"timestamp" format:"date-time"`
    Email     string    `json:"email" format:"email"`
    IP        string    `json:"ip" format:"ipv4"`
}
```

## Optional vs Required

**Default: All fields required**

Make optional with `omitempty` or `omitzero`:
```go
type Input struct {
    Required string  `json:"required"`              // Required
    Optional *string `json:"optional,omitempty"`    // Optional
    OptInt   int     `json:"opt_int,omitzero"`      // Optional, zero value allowed
}
```

**Override with tags:**
```go
type Input struct {
    MustHave string  `json:"must" required:"true"`   // Required even with omitempty
    CanSkip  string  `json:"skip" required:"false"`  // Optional even without omitempty
}
```

## Nullable Fields

**Scalars:** Use pointer + no omitempty
```go
type Input struct {
    Count *int `json:"count"` // nullable, can be null or number
}
```

**Arrays:** Set global default or use tag
```go
huma.DefaultArrayNullable = true // All arrays nullable

type Input struct {
    Items []string `json:"items" nullable:"true"` // This array is nullable
}
```

**Structs:** Add zero-width field
```go
type Address struct {
    _ struct{} `nullable:"true"`
    Street string `json:"street"`
    City   string `json:"city"`
}
```

## Enums

```go
type Status string

type Order struct {
    Status Status `json:"status" enum:"pending,processing,completed,cancelled"`
}
```

## Documentation Tags

```go
type Product struct {
    Name  string  `json:"name" doc:"Product name" example:"Widget"`
    Price float64 `json:"price" doc:"Price in USD" example:"19.99" default:"0"`
}
```

## Hidden/ReadOnly/WriteOnly

```go
type User struct {
    Password    string    `json:"password" writeOnly:"true"`  // Only in requests
    CreatedAt   time.Time `json:"created_at" readOnly:"true"` // Only in responses
    InternalID  int       `json:"-" hidden:"true"`            // Never in OpenAPI
}
```

## Deprecated Fields

```go
type Product struct {
    OldField string `json:"old_field" deprecated:"true" doc:"Use new_field instead"`
    NewField string `json:"new_field"`
}
```

## Dependent Required

```go
type Payment struct {
    Method      string  `json:"method" enum:"card,bank"`
    CardNumber  *string `json:"card_number,omitempty" dependentRequired:"method=card"`
    BankAccount *string `json:"bank_account,omitempty" dependentRequired:"method=bank"`
}
```

## Custom Time Formats

```go
type Event struct {
    UnixTime int64 `json:"unix_time" doc:"Unix timestamp" example:"1640000000"`

    // Custom format
    CustomDate time.Time `json:"custom" timeFormat:"2006-01-02"`
}
```

## Validation Tips

1. **Be specific** - Use tight constraints for better validation
2. **Provide examples** - Help API consumers understand expected values
3. **Document patterns** - Add `patternDescription` for complex regex
4. **Use formats** - Leverage built-in format validators
5. **Set reasonable limits** - Prevent abuse with minLength/maxLength/minimum/maximum
