# mocky

A Go HTTP server that configures API routes from YAML.

Each route can define:

- a static response;
- a built-in handler provided by the server itself;
- an external executable that receives the request through `stdin` and returns a response through `stdout`.

## Config format

```yaml
server:
  address: ":8080"

routes:
  - path: /hello
    method: GET
    response:
      status: 200
      headers:
        Content-Type: application/json
      body:
        message: hello

  - path: /process
    method: POST
    builtin:
      name: echo

  - path: /external-process
    method: POST
    exec:
      command: ./scripts/custom-handler.sh
      args: []
      dir: .
      timeout_seconds: 5
      pass_body: true
      env:
        DEMO_MODE: "1"

  - path: /jobs
    method: POST
    async: true
    async_response:
      status: 202
      headers:
        Content-Type: application/json
      body:
        status: accepted
        queued: true
    exec:
      command: ./scripts/echo-handler.sh
      timeout_seconds: 30
      pass_body: true
```

## `exec` contract

The server sends JSON to the process through `stdin`:

```json
{
  "method": "POST",
  "path": "/process",
  "query": {"debug":["1"]},
  "headers": {"Content-Type":["application/json"]},
  "body": "{\"foo\":\"bar\"}",
  "body_base64": "eyJmb28iOiJiYXIifQ",
  "remote_addr": "127.0.0.1:12345",
  "content_type": "application/json"
}
```

The process must return JSON to `stdout`:

```json
{
  "status": 200,
  "headers": {
    "Content-Type": "application/json"
  },
  "body": {
    "result": "ok"
  }
}
```

## Asynchronous handling

Background execution can be enabled for routes with `exec`:

```yaml
- path: /jobs
  method: POST
  async: true
  async_response:
    status: 202
    headers:
      Content-Type: application/json
    body:
      status: accepted
      queued: true
  builtin:
    name: echo
```

Behavior:

- the client immediately receives `202 Accepted` or the response from `async_response`;
- the built-in handler or external process starts in the background;
- the final result is not returned to the client, but errors are written to the server log;
- `async: true` is supported for both `builtin` and `exec` routes.

## Built-in handlers

Currently there is one built-in handler:

```yaml
- path: /echo
  method: POST
  builtin:
    name: echo
```

`builtin: echo` returns JSON with the request method, path, and body. It is the default demo handler and does not depend on Python or external scripts.

## Running

```bash
go run ./cmd/mocky --config ./config.yaml
```

Or in daemon mode:

```bash
go run ./cmd/mocky --config ./config.yaml --daemon
```

Show built-in help and a full config example:

```bash
go run ./cmd/mocky --help
```

## Build and install

Build the binary into `build/`:

```bash
make build
./build/mocky --config ./config.yaml
```

Install into the user's Go bin directory:

```bash
make install-local
mocky --config ./config.yaml
```

Install system-wide into `/usr/local/bin`:

```bash
make build
sudo make install
mocky --config /absolute/path/to/config.yaml
```

By default, `make install` places the binary into `/usr/local/bin`, but the prefix can be overridden:

```bash
make build
make install PREFIX=/opt/mocky
```

After startup:

- `GET /health` returns static JSON;
- `POST /echo` is handled by the built-in `builtin: echo`;
- `POST /jobs` immediately returns `202`, while the built-in handler runs in the background;
- external `exec` handlers can still be used when needed.

## Current limitations

- route matching is based on exact `path` matches;
- the request body is passed to an external process only when `pass_body: true`;
- if an external process fails, the server returns `500`;
- `--daemon` is currently implemented by starting a background process with `stdin`, `stdout`, and `stderr` redirected to `/dev/null`;
- in async mode, the result of `builtin` or `exec` execution is only logged and is not returned to the client.
