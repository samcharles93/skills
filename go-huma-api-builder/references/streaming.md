# HUMA Streaming & Server-Sent Events

## Basic Streaming

Use `huma.StreamResponse` for streaming responses:

```go
func streamHandler(ctx context.Context, input *struct{}) (*huma.StreamResponse, error) {
    return &huma.StreamResponse{
        Body: func(ctx huma.Context) {
            writer := ctx.BodyWriter()

            for i := 0; i < 10; i++ {
                fmt.Fprintf(writer, "Chunk %d\n", i)

                // Flush if possible
                if f, ok := writer.(http.Flusher); ok {
                    f.Flush()
                }

                time.Sleep(100 * time.Millisecond)
            }
        },
    }, nil
}

huma.Register(api, huma.Operation{
    OperationID: "stream",
    Method:      http.MethodGet,
    Path:        "/stream",
}, streamHandler)
```

## Server-Sent Events (SSE)

```go
import "github.com/danielgtaylor/huma/v2/sse"

type ProgressEvent struct {
    Progress int    `json:"progress"`
    Message  string `json:"message"`
}

type CompletionEvent struct {
    Result string `json:"result"`
}

func sseHandler(ctx context.Context, input *struct{}, send sse.Sender) {
    // Send progress updates
    for i := 0; i <= 100; i += 10 {
        if err := send.Data(ProgressEvent{
            Progress: i,
            Message:  fmt.Sprintf("Processing... %d%%", i),
        }); err != nil {
            return // Client disconnected
        }
        time.Sleep(100 * time.Millisecond)
    }

    // Send completion
    send.Data(CompletionEvent{
        Result: "Done!",
    })
}

sse.Register(api, huma.Operation{
    OperationID: "progress",
    Method:      http.MethodGet,
    Path:        "/progress",
}, map[string]any{
    "progress":   ProgressEvent{},
    "completion": CompletionEvent{},
}, sseHandler)
```

## SSE with Named Events

```go
func sseHandlerWithNames(ctx context.Context, input *struct{}, send sse.Sender) {
    // Send with event type
    send.Event("update", ProgressEvent{Progress: 50})

    // Send without event type (uses "message" by default)
    send.Data(ProgressEvent{Progress: 75})

    // Send with event type
    send.Event("done", CompletionEvent{Result: "Complete"})
}
```

## SSE with Retry and ID

```go
func sseHandlerWithRetry(ctx context.Context, input *struct{}, send sse.Sender) {
    // Set retry interval (milliseconds)
    send.Retry(3000)

    eventID := 1
    for {
        select {
        case data := <-dataChannel:
            // Send with ID for resumption
            send.ID(fmt.Sprintf("%d", eventID))
            send.Data(data)
            eventID++

        case <-ctx.Done():
            return
        }
    }
}
```

## Streaming JSON Lines

```go
type Item struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

func jsonLinesHandler(ctx context.Context, input *struct{}) (*huma.StreamResponse, error) {
    return &huma.StreamResponse{
        Body: func(ctx huma.Context) {
            ctx.SetHeader("Content-Type", "application/x-ndjson")

            writer := ctx.BodyWriter()
            encoder := json.NewEncoder(writer)

            items := []Item{
                {ID: "1", Name: "Item 1"},
                {ID: "2", Name: "Item 2"},
                {ID: "3", Name: "Item 3"},
            }

            for _, item := range items {
                encoder.Encode(item)

                if f, ok := writer.(http.Flusher); ok {
                    f.Flush()
                }
            }
        },
    }, nil
}
```

## Streaming Large Files

```go
func downloadHandler(ctx context.Context, input *DownloadInput) (*huma.StreamResponse, error) {
    file, err := os.Open(input.FilePath)
    if err != nil {
        return nil, huma.Error404NotFound("File not found")
    }

    return &huma.StreamResponse{
        Body: func(ctx huma.Context) {
            defer file.Close()

            ctx.SetHeader("Content-Type", "application/octet-stream")
            ctx.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(input.FilePath)))

            // Stream file in chunks
            io.Copy(ctx.BodyWriter(), file)
        },
    }, nil
}
```

## Streaming with Context Cancellation

```go
func streamHandlerWithCancel(ctx context.Context, input *struct{}) (*huma.StreamResponse, error) {
    return &huma.StreamResponse{
        Body: func(humaCtx huma.Context) {
            writer := humaCtx.BodyWriter()

            ticker := time.NewTicker(100 * time.Millisecond)
            defer ticker.Stop()

            for {
                select {
                case <-ctx.Done():
                    // Client disconnected or request cancelled
                    return

                case <-ticker.C:
                    if _, err := fmt.Fprintf(writer, "data: tick\n\n"); err != nil {
                        return
                    }

                    if f, ok := writer.(http.Flusher); ok {
                        f.Flush()
                    }
                }
            }
        },
    }, nil
}
```

## Chat Completion Streaming

```go
type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatInput struct {
    Body struct {
        Messages []ChatMessage `json:"messages"`
        Stream   bool          `json:"stream"`
    }
}

func chatHandler(ctx context.Context, input *ChatInput) (*huma.StreamResponse, error) {
    if !input.Body.Stream {
        // Non-streaming response
        return &huma.StreamResponse{
            Body: func(humaCtx huma.Context) {
                response := generateResponse(input.Body.Messages)
                json.NewEncoder(humaCtx.BodyWriter()).Encode(response)
            },
        }, nil
    }

    // Streaming response
    return &huma.StreamResponse{
        Body: func(humaCtx huma.Context) {
            humaCtx.SetHeader("Content-Type", "text/event-stream")
            writer := humaCtx.BodyWriter()

            for token := range generateStreamingResponse(ctx, input.Body.Messages) {
                fmt.Fprintf(writer, "data: %s\n\n", token)

                if f, ok := writer.(http.Flusher); ok {
                    f.Flush()
                }

                select {
                case <-ctx.Done():
                    return
                default:
                }
            }

            fmt.Fprintf(writer, "data: [DONE]\n\n")
        },
    }, nil
}
```

## SSE Client Example

```go
func consumeSSE(url string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    reader := bufio.NewReader(resp.Body)

    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                break
            }
            return err
        }

        line = strings.TrimSpace(line)

        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimPrefix(line, "data: ")
            fmt.Println("Received:", data)
        }
    }

    return nil
}
```

## Streaming Best Practices

1. **Set appropriate headers** - `Content-Type`, `Cache-Control: no-cache`, etc.
2. **Flush regularly** - Ensure data is sent to client immediately
3. **Handle disconnections** - Check context cancellation
4. **Use appropriate formats** - SSE for events, NDJSON for structured data
5. **Set timeouts carefully** - Long-running streams need longer timeouts
6. **Document event types** - Clearly specify what events clients can expect
7. **Provide retry mechanism** - For SSE, set appropriate retry intervals
8. **Test with slow connections** - Ensure buffering doesn't cause issues

## Common Patterns

### Progress Updates
```go
func longRunningTask(ctx context.Context, input *struct{}, send sse.Sender) {
    steps := []string{"Preparing", "Processing", "Finalizing"}

    for i, step := range steps {
        send.Data(ProgressEvent{
            Step:     step,
            Progress: int(float64(i+1) / float64(len(steps)) * 100),
        })

        // Do work
        doWork(ctx, step)
    }

    send.Data(CompletionEvent{Status: "success"})
}
```

### Real-time Logs
```go
func streamLogs(ctx context.Context, input *LogsInput, send sse.Sender) {
    tailCmd := exec.CommandContext(ctx, "tail", "-f", input.LogFile)
    stdout, _ := tailCmd.StdoutPipe()
    tailCmd.Start()

    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        send.Data(LogEntry{
            Timestamp: time.Now(),
            Message:   scanner.Text(),
        })
    }
}
```

### Chunked Data Export
```go
func exportData(ctx context.Context, input *ExportInput) (*huma.StreamResponse, error) {
    return &huma.StreamResponse{
        Body: func(humaCtx huma.Context) {
            humaCtx.SetHeader("Content-Type", "text/csv")
            humaCtx.SetHeader("Content-Disposition", "attachment; filename=export.csv")

            writer := csv.NewWriter(humaCtx.BodyWriter())

            // Stream rows from database
            rows, _ := db.Query("SELECT * FROM large_table")
            defer rows.Close()

            for rows.Next() {
                var row []string
                rows.Scan(&row)
                writer.Write(row)
                writer.Flush()

                if f, ok := humaCtx.BodyWriter().(http.Flusher); ok {
                    f.Flush()
                }
            }
        },
    }, nil
}
```
