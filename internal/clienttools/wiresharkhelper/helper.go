package wiresharkhelper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const DefaultAddress = "127.0.0.1:38765"

var ErrWiresharkNotFound = errors.New("wireshark executable not found")

type Launcher interface {
	Available() bool
	Launch(string) error
}

type Handler struct {
	AllowedOrigin string
	Version       string
	Launcher      Launcher
}

func (h Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/health" {
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Private-Network", "true")
		if request.Method == http.MethodOptions {
			writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.Method != http.MethodGet {
			writeProblem(writer, http.StatusMethodNotAllowed, "method_not_allowed", "GET is required")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"status":              "ok",
			"version":             h.Version,
			"allowed_origin":      h.AllowedOrigin,
			"wireshark_available": h.Launcher != nil && h.Launcher.Available(),
		})
		return
	}
	if request.URL.Path != "/launch" {
		http.NotFound(writer, request)
		return
	}
	if !h.allowOrigin(writer, request) {
		return
	}
	if request.Method == http.MethodOptions {
		writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		writer.Header().Set("Access-Control-Allow-Private-Network", "true")
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	if request.Method != http.MethodPost {
		writeProblem(writer, http.StatusMethodNotAllowed, "method_not_allowed", "POST is required")
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 16<<10)
	var body struct {
		StreamURL string `json:"stream_url"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeProblem(writer, http.StatusBadRequest, "invalid_request", "stream_url is required")
		return
	}
	if err := validateStreamURL(body.StreamURL, h.AllowedOrigin); err != nil {
		writeProblem(writer, http.StatusBadRequest, "invalid_stream_url", err.Error())
		return
	}
	if h.Launcher == nil || !h.Launcher.Available() {
		writeProblem(writer, http.StatusFailedDependency, "wireshark_not_found", "Wireshark is not installed or could not be found")
		return
	}
	if err := h.Launcher.Launch(body.StreamURL); err != nil {
		status := http.StatusBadGateway
		code := "wireshark_launch_failed"
		if errors.Is(err, ErrWiresharkNotFound) {
			status = http.StatusFailedDependency
			code = "wireshark_not_found"
		}
		writeProblem(writer, status, code, err.Error())
		return
	}
	writeJSON(writer, http.StatusAccepted, map[string]any{"status": "launched"})
}

func (h Handler) allowOrigin(writer http.ResponseWriter, request *http.Request) bool {
	origin := request.Header.Get("Origin")
	if h.AllowedOrigin == "" || origin != h.AllowedOrigin {
		writeProblem(writer, http.StatusForbidden, "origin_not_allowed", "restart the helper with -allow-origin set to this NetLab address")
		return false
	}
	writer.Header().Set("Access-Control-Allow-Origin", h.AllowedOrigin)
	writer.Header().Set("Access-Control-Allow-Private-Network", "true")
	writer.Header().Set("Vary", "Origin")
	return true
}

func validateStreamURL(value, allowedOrigin string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("stream_url must be an absolute HTTP or HTTPS URL")
	}
	origin := parsed.Scheme + "://" + parsed.Host
	if allowedOrigin == "" || origin != allowedOrigin {
		return fmt.Errorf("stream_url origin must match the configured NetLab origin")
	}
	if !strings.HasPrefix(parsed.Path, "/api/v1/captures/") || !strings.HasSuffix(parsed.Path, "/stream") {
		return fmt.Errorf("stream_url must reference a NetLab capture stream")
	}
	return nil
}

type RealLauncher struct {
	executable string
	client     *http.Client
}

func NewRealLauncher(explicit string) *RealLauncher {
	return &RealLauncher{
		executable: findWireshark(explicit),
		client: &http.Client{Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		}},
	}
}

func (l *RealLauncher) Available() bool { return l != nil && l.executable != "" }

func (l *RealLauncher) Launch(streamURL string) error {
	if !l.Available() {
		return ErrWiresharkNotFound
	}
	response, err := l.client.Get(streamURL)
	if err != nil {
		return fmt.Errorf("open capture stream: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("capture stream returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	command := exec.Command(l.executable, "-k", "-i", "-")
	command.Stdin = response.Body
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err = command.Start(); err != nil {
		response.Body.Close()
		return fmt.Errorf("start Wireshark: %w", err)
	}
	go func() {
		_ = command.Wait()
		_ = response.Body.Close()
	}()
	return nil
}

func findWireshark(explicit string) string {
	if explicit != "" {
		if path, err := exec.LookPath(explicit); err == nil {
			return path
		}
		if info, err := os.Stat(explicit); err == nil && !info.IsDir() {
			return explicit
		}
		return ""
	}
	for _, name := range []string{"wireshark", "Wireshark.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Wireshark", "Wireshark.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Wireshark", "Wireshark.exe"),
		}
	case "darwin":
		candidates = []string{"/Applications/Wireshark.app/Contents/MacOS/Wireshark"}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func ValidateListenAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("helper must listen on a loopback address")
	}
	return nil
}

func writeProblem(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, map[string]string{"code": code, "message": message})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func Serve(ctx context.Context, address string, handler http.Handler) error {
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
