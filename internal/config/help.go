package config

const HelpText = `mocky - HTTP mock server driven by YAML config

Usage:
  go build -o ./build/mocky ./cmd/mocky
  ./build/mocky --config ./config.yaml
  go install ./cmd/mocky
  mocky --config ./config.yaml
  mocky --config ./config.yaml --daemon
  mocky --help

Flags:
  --config string
      Path to YAML config file. Default: ./config.yaml
  --daemon
      Run the server in background and return control to the shell.
  --help
      Show this help, including a full config example.

Install:
  1. Build binary into the project:
     go build -o ./build/mocky ./cmd/mocky
  2. Run built binary:
     ./build/mocky --config ./config.yaml
  3. Install into Go bin directory:
     go install ./cmd/mocky
  4. Or use Makefile:
     make build
     make install-local
     sudo make install

Config example:
server:
  address: ":8080"

routes:
  - path: /health
    method: GET
    response:
      status: 200
      headers:
        Content-Type: application/json
        X-Mock-Source: static
      body:
        status: ok
        service: mocky

  - path: /hello-text
    method: GET
    response:
      status: 200
      headers:
        Content-Type: text/plain; charset=utf-8
      body: hello from mocky

  - path: /echo
    method: POST
    builtin:
      name: echo

  - path: /jobs
    method: POST
    async: true
    async_response:
      status: 202
      headers:
        Content-Type: application/json
      body:
        status: accepted
        mode: async
    builtin:
      name: echo

  - path: /process
    method: POST
    exec:
      command: ./scripts/custom-handler.sh
      timeout_seconds: 5
      pass_body: true

How exec works:
  1. Server starts the configured command.
  2. Request data is passed to stdin as JSON:
     {
       "method": "POST",
       "path": "/echo",
       "query": {"debug":["1"]},
       "headers": {"Content-Type":["application/json"]},
       "body": "{\"hello\":\"world\"}",
       "body_base64": "eyJoZWxsbyI6IndvcmxkIn0",
       "remote_addr": "127.0.0.1:12345",
       "content_type": "application/json"
     }
  3. Command must write JSON to stdout:
     {
       "status": 200,
       "headers": {
         "Content-Type": "application/json"
       },
       "body": {
         "message": "handled"
       }
     }
`
