# Docker Integration Patterns in Go

Use a maintained client library, for example `github.com/fsouza/go-dockerclient`
(shown below) or the official `github.com/docker/docker/client`. Create one
client at startup and share it; its underlying `http.Transport` already pools
connections, so no wrapper struct is needed.

## Client Setup

```go
client, err := docker.NewClientFromEnv() // honours DOCKER_HOST and TLS env vars
if err != nil {
    return fmt.Errorf("create docker client: %w", err)
}
```

Readiness probe: `client.PingWithContext(ctx)`, and register it with the health
checker in `resilience.md`.

## Buffer Pooling

Pool `bytes.Buffer` with `sync.Pool`; do not hand-roll a circular buffer for
capturing output:

```go
var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func execInContainer(ctx context.Context, client *docker.Client, id string, cmd []string) (stdout, stderr string, err error) {
    outBuf := bufPool.Get().(*bytes.Buffer)
    errBuf := bufPool.Get().(*bytes.Buffer)
    defer func() {
        outBuf.Reset()
        errBuf.Reset()
        bufPool.Put(outBuf)
        bufPool.Put(errBuf)
    }()

    exec, err := client.CreateExec(docker.CreateExecOptions{
        Container: id, Cmd: cmd, AttachStdout: true, AttachStderr: true, Context: ctx,
    })
    if err != nil {
        return "", "", fmt.Errorf("create exec: %w", err)
    }
    if err := client.StartExec(exec.ID, docker.StartExecOptions{
        OutputStream: outBuf, ErrorStream: errBuf, Context: ctx,
    }); err != nil {
        return "", "", fmt.Errorf("start exec: %w", err)
    }
    return outBuf.String(), errBuf.String(), nil
}
```

## Container Lifecycle

```go
container, err := client.CreateContainer(docker.CreateContainerOptions{
    Config:     &docker.Config{Image: image, Cmd: cmd, Env: envSlice}, // env as "k=v" slice
    HostConfig: &docker.HostConfig{AutoRemove: true},
    Context:    ctx,
})
if err != nil {
    return "", fmt.Errorf("create container: %w", err)
}
if err := client.StartContainer(container.ID, nil); err != nil {
    // AutoRemove cleans up, but force-remove anyway to be safe
    _ = client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID, Force: true})
    return "", fmt.Errorf("start container: %w", err)
}
exitCode, err := client.WaitContainer(container.ID) // blocks; cancel via ctx-aware remove
```

Prefer `AutoRemove` so failed starts do not leak containers. `WaitContainer`
blocks; when you need cancellation, select on `ctx.Done()` and force-remove.

## Stats, Events, Images, Labels

- **Stats:** `client.Stats(docker.StatsOptions{ID: id, Stream: false, Context: ctx})`
  delivers on a channel; compute CPU percent from the pre/post usage deltas.
- **Events:** `client.AddEventListener(ch)` / `client.RemoveEventListener(ch)`,
  then `select` on the channel and `ctx.Done()`.
- **Images:** `client.InspectImage(image)` to test existence, `client.PullImage(...)`
  to fetch; wrap the pair as `EnsureImage`.
- **Labels:** `client.ListContainers(...)` exposes `container.Labels`. Keep the
  parsing next to the component that owns the label scheme, not in the client
  wrapper.

For integration tests that need a real daemon, see the Docker-based pattern in
`testing.md` and run them behind a build tag (`//go:build integration`).
