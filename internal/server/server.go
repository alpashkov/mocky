package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"mocky/internal/config"
)

func New(cfg *config.Config, configDir string) (http.Handler, error) {
	mux := http.NewServeMux()
	executor := NewExecutor(configDir)

	for _, route := range cfg.Routes {
		route := route

		mux.HandleFunc(route.Path, func(w http.ResponseWriter, r *http.Request) {
			if route.Method != "" && !strings.EqualFold(route.Method, r.Method) {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var err error
			switch {
			case route.Response != nil:
				err = writeStaticResponse(w, route.Response)
			case route.Async:
				err = executor.ExecuteAsync(w, r, route)
			case route.Builtin != nil || route.Exec != nil:
				err = executor.Execute(w, r, route)
			default:
				err = errors.New("route action is not configured")
			}

			if err != nil {
				log.Printf("request %s %s failed: %v", r.Method, r.URL.Path, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
	}

	return mux, nil
}

func writeStaticResponse(w http.ResponseWriter, response *config.StaticResponse) error {
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}

	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}

	switch body := response.Body.(type) {
	case nil:
		setDefaultContentType(w, "application/json")
		w.WriteHeader(status)
		return nil
	case string:
		setDefaultContentType(w, "text/plain; charset=utf-8")
		w.WriteHeader(status)
		_, err := io.WriteString(w, body)
		return err
	default:
		setDefaultContentType(w, "application/json")
		w.WriteHeader(status)
		return json.NewEncoder(w).Encode(body)
	}
}

func writeExecOutput(w http.ResponseWriter, result *ExecOutput) error {
	status := result.Status
	if status == 0 {
		status = http.StatusOK
	}

	for key, value := range result.Headers {
		w.Header().Set(key, value)
	}

	switch body := result.Body.(type) {
	case nil:
		setDefaultContentType(w, "application/json")
		w.WriteHeader(status)
		return nil
	case string:
		setDefaultContentType(w, "text/plain; charset=utf-8")
		w.WriteHeader(status)
		_, err := io.WriteString(w, body)
		return err
	default:
		setDefaultContentType(w, "application/json")
		w.WriteHeader(status)
		return json.NewEncoder(w).Encode(body)
	}
}

func setDefaultContentType(w http.ResponseWriter, value string) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", value)
	}
}
