package wiresharkhelper

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeLauncher struct {
	available bool
	streamURL string
	err       error
}

func (f *fakeLauncher) Available() bool { return f.available }
func (f *fakeLauncher) Launch(streamURL string) error {
	f.streamURL = streamURL
	return f.err
}

func TestLaunchRequiresConfiguredOriginAndCaptureStream(t *testing.T) {
	launcher := &fakeLauncher{available: true}
	handler := Handler{AllowedOrigin: "http://netlab.test", Version: "test", Launcher: launcher}
	request := httptest.NewRequest(http.MethodPost, "/launch", bytes.NewBufferString(`{"stream_url":"http://netlab.test/api/v1/captures/capture-1/stream"}`))
	request.Header.Set("Origin", "http://netlab.test")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if launcher.streamURL != "http://netlab.test/api/v1/captures/capture-1/stream" {
		t.Fatalf("stream=%q", launcher.streamURL)
	}
}

func TestLaunchReportsMissingWireshark(t *testing.T) {
	handler := Handler{AllowedOrigin: "http://netlab.test", Launcher: &fakeLauncher{}}
	request := httptest.NewRequest(http.MethodPost, "/launch", bytes.NewBufferString(`{"stream_url":"http://netlab.test/api/v1/captures/capture-1/stream"}`))
	request.Header.Set("Origin", "http://netlab.test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusFailedDependency || !bytes.Contains(response.Body.Bytes(), []byte("wireshark_not_found")) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestHealthIsDiscoverableAcrossOrigins(t *testing.T) {
	handler := Handler{AllowedOrigin: "http://netlab.test", Version: "1.2.3", Launcher: &fakeLauncher{available: true}}
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("status=%d headers=%v", response.Code, response.Header())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"wireshark_available":true`)) {
		t.Fatalf("body=%s", response.Body.String())
	}
}

func TestHealthAllowsPrivateNetworkPreflight(t *testing.T) {
	handler := Handler{AllowedOrigin: "http://netlab.test", Launcher: &fakeLauncher{available: true}}
	request := httptest.NewRequest(http.MethodOptions, "/health", nil)
	request.Header.Set("Origin", "http://netlab.test")
	request.Header.Set("Access-Control-Request-Private-Network", "true")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Private-Network") != "true" {
		t.Fatalf("status=%d headers=%v", response.Code, response.Header())
	}
}

func TestListenAddressMustBeLoopback(t *testing.T) {
	if err := ValidateListenAddress("127.0.0.1:38765"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateListenAddress("0.0.0.0:38765"); err == nil {
		t.Fatal("non-loopback address accepted")
	}
}
