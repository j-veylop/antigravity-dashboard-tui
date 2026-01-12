package quota

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRefreshAccessToken(t *testing.T) {
	tests := []struct {
		name         string
		refreshToken string
		transport    http.RoundTripper
		wantErr      bool
	}{
		{
			name:         "Success",
			refreshToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					body, _ := json.Marshal(TokenResponse{AccessToken: "new"})
					return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
				},
			},
			wantErr: false,
		},
		{
			name:         "EmptyToken",
			refreshToken: "",
			wantErr:      true,
		},
		{
			name:         "HTTPError",
			refreshToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("net error")
				},
			},
			wantErr: true,
		},
		{
			name:         "StatusError",
			refreshToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader("bad request"))}, nil
				},
			},
			wantErr: true,
		},
		{
			name:         "JSONError",
			refreshToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("invalid json"))}, nil
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: tt.transport}
			if tt.transport == nil {
				client = nil
			}
			_, err := RefreshAccessToken(client, tt.refreshToken, "cid", "csec")
			if (err != nil) != tt.wantErr {
				t.Errorf("RefreshAccessToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetchQuota(t *testing.T) {
	tests := []struct {
		name        string
		accessToken string
		transport   http.RoundTripper
		wantErr     bool
	}{
		{
			name:        "Success",
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					body, _ := json.Marshal(fetchModelsResponse{})
					return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
				},
			},
			wantErr: false,
		},
		{
			name:        "EmptyToken",
			accessToken: "",
			wantErr:     true,
		},
		{
			name:        "AllEndpointsFail",
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("fail")
				},
			},
			wantErr: true,
		},
		{
			name:        "Unauthorized",
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("unauthorized"))}, nil
				},
			},
			wantErr: true,
		},
		{
			name:        "StatusError",
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("error"))}, nil
				},
			},
			wantErr: true,
		},
		{
			name:        "JSONError",
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("bad json"))}, nil
				},
			},
			wantErr: true,
		},
		{
			name:        "ParseTimeError", // This one doesn't error the function, but logs error. Coverage!
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					resp := fetchModelsResponse{
						Models: map[string]struct {
							DisplayName string `json:"displayName"`
							QuotaInfo   struct {
								ResetTime         string  `json:"resetTime"`
								RemainingFraction float64 `json:"remainingFraction"`
							} `json:"quotaInfo"`
						}{
							"test": {
								QuotaInfo: struct {
									ResetTime         string  `json:"resetTime"`
									RemainingFraction float64 `json:"remainingFraction"`
								}{
									ResetTime: "invalid-time",
								},
							},
						},
					}
					body, _ := json.Marshal(resp)
					return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: tt.transport}
			if tt.transport == nil {
				client = nil
			}
			_, err := FetchQuota(client, tt.accessToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchQuota() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetchUserInfo(t *testing.T) {
	tests := []struct {
		name        string
		accessToken string
		transport   http.RoundTripper
		wantErr     bool
	}{
		{
			name:        "Success",
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					body, _ := json.Marshal(UserInfo{Email: "test@example.com"})
					return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
				},
			},
			wantErr: false,
		},
		{
			name:        "EmptyToken",
			accessToken: "",
			wantErr:     true,
		},
		{
			name:        "HTTPError",
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("net error")
				},
			},
			wantErr: true,
		},
		{
			name:        "StatusError",
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader("bad request"))}, nil
				},
			},
			wantErr: true,
		},
		{
			name:        "JSONError",
			accessToken: "valid",
			transport: &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("bad json"))}, nil
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: tt.transport}
			if tt.transport == nil {
				client = nil
			}
			_, err := FetchUserInfo(client, tt.accessToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchUserInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCachedToken_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		token *CachedToken
		want  bool
	}{
		{"Nil", nil, false},
		{"EmptyAccess", &CachedToken{}, false},
		{"Expired", &CachedToken{AccessToken: "t", ExpiresAt: time.Now().Add(-1 * time.Minute)}, false},
		{"Valid", &CachedToken{AccessToken: "t", ExpiresAt: time.Now().Add(10 * time.Minute)}, true},
		{"BufferEdge", &CachedToken{AccessToken: "t", ExpiresAt: time.Now().Add(4 * time.Minute)}, false}, // < 5 min buffer
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsValid(); got != tt.want {
				t.Errorf("CachedToken.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
