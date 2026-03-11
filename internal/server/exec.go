package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mocky/internal/config"
)

type Executor struct {
	configDir string
}

type ExecInput struct {
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Query       map[string][]string `json:"query"`
	Headers     map[string][]string `json:"headers"`
	Body        string              `json:"body,omitempty"`
	BodyBase64  string              `json:"body_base64,omitempty"`
	RemoteAddr  string              `json:"remote_addr"`
	ContentType string              `json:"content_type,omitempty"`
}

type ExecOutput struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

type requestData struct {
	input   ExecInput
	rawBody []byte
}

func NewExecutor(configDir string) *Executor {
	return &Executor{configDir: configDir}
}

func (e *Executor) Execute(w http.ResponseWriter, r *http.Request, route config.Route) error {
	data, err := readRequest(r, route)
	if err != nil {
		return err
	}

	result, err := e.run(r.Context(), data, route)
	if err != nil {
		return err
	}

	return writeExecOutput(w, result)
}

func (e *Executor) ExecuteAsync(w http.ResponseWriter, r *http.Request, route config.Route) error {
	data, err := readRequest(r, route)
	if err != nil {
		return err
	}

	go func() {
		if _, err := e.run(context.Background(), data, route); err != nil {
			log.Printf("async request %s %s failed: %v", data.input.Method, data.input.Path, err)
			return
		}

		log.Printf("async request %s %s completed", data.input.Method, data.input.Path)
	}()

	if route.AsyncResponse != nil {
		return writeStaticResponse(w, route.AsyncResponse)
	}

	return writeStaticResponse(w, &config.StaticResponse{
		Status: http.StatusAccepted,
		Body: map[string]any{
			"status": "accepted",
			"mode":   "async",
		},
	})
}

func (e *Executor) run(parent context.Context, data *requestData, route config.Route) (*ExecOutput, error) {
	if route.Builtin != nil {
		return runBuiltin(data, route.Builtin)
	}
	if route.Exec == nil {
		return nil, errors.New("route has no executable action")
	}

	action := route.Exec
	timeout := time.Duration(action.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.commandPath(action), action.Args...)
	cmd.Dir = e.commandDir(action)
	cmd.Env = append(os.Environ(), flattenEnv(action.Env)...)

	payload, err := json.Marshal(data.input)
	if err != nil {
		return nil, fmt.Errorf("marshal exec input: %w", err)
	}

	cmd.Stdin = bytes.NewReader(payload)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("run command: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("run command: %w", err)
	}

	var result ExecOutput
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("decode command output: %w", err)
	}

	return &result, nil
}

func runBuiltin(data *requestData, action *config.BuiltinAction) (*ExecOutput, error) {
	switch action.Name {
	case "echo":
		return &ExecOutput{
			Status: http.StatusOK,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]any{
				"message":       "handled by builtin echo",
				"method":        data.input.Method,
				"path":          data.input.Path,
				"received_body": string(data.rawBody),
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown builtin handler: %s", action.Name)
	}
}

func (e *Executor) commandPath(action *config.ExecAction) string {
	if filepath.IsAbs(action.Command) || action.Dir != "" {
		return action.Command
	}

	return filepath.Join(e.configDir, action.Command)
}

func (e *Executor) commandDir(action *config.ExecAction) string {
	if action.Dir == "" {
		return e.configDir
	}
	if filepath.IsAbs(action.Dir) {
		return action.Dir
	}

	return filepath.Join(e.configDir, action.Dir)
}

func readRequest(r *http.Request, route config.Route) (*requestData, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}

	input := ExecInput{
		Method:      r.Method,
		Path:        r.URL.Path,
		Query:       r.URL.Query(),
		Headers:     r.Header.Clone(),
		RemoteAddr:  r.RemoteAddr,
		ContentType: r.Header.Get("Content-Type"),
	}

	if shouldIncludeBody(route) {
		input.Body = string(body)
		input.BodyBase64 = base64.RawURLEncoding.EncodeToString(body)
	}

	return &requestData{
		input:   input,
		rawBody: body,
	}, nil
}

func flattenEnv(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}

	items := make([]string, 0, len(values))
	for key, value := range values {
		items = append(items, fmt.Sprintf("%s=%s", key, value))
	}

	return items
}

func shouldIncludeBody(route config.Route) bool {
	if route.Builtin != nil {
		return true
	}

	return route.Exec != nil && route.Exec.PassBody
}
