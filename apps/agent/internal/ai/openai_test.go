package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/internal/protocol"
)

const sampleAnalysisJSON = `{
  "pageType": "article",
  "summary": "A short summary",
  "keyPoints": [],
  "concepts": [],
  "conclusions": [],
  "actions": [],
  "questions": [],
  "tags": []
}`

func testPacket(image any) protocol.ContentPacket {
	packet := protocol.ContentPacket{
		ProtocolVersion: protocol.Version,
		CaptureID:       "capture-1",
		CapturedAt:      "2026-08-13T10:00:00Z",
		Source: protocol.Source{
			URL:   "https://example.com/article",
			Title: "Example",
		},
		Content: protocol.Content{Markdown: "Hello world"},
	}
	if image != nil {
		packet.Metadata = map[string]any{"image": image}
	}
	return packet
}

type capturedChatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"messages"`
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (*OpenAICompatible, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewOpenAICompatible(server.URL, "sk-test-secret-key", "google/diffusiongemma-26b-a4b-it", 12000, 5*time.Second)
	client.now = func() time.Time { return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC) }
	return client, server
}

func writeChatCompletion(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"content": content}},
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func decodePostedChat(t *testing.T, body []byte) capturedChatRequest {
	t.Helper()
	var req capturedChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("decode posted chat body: %v\n%s", err, body)
	}
	return req
}

func userContent(t *testing.T, req capturedChatRequest) json.RawMessage {
	t.Helper()
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			return msg.Content
		}
	}
	t.Fatal("missing user message")
	return nil
}

func TestAnalyzeTextOnlyKeepsStringContent(t *testing.T) {
	var posted []byte
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var err error
		posted, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		writeChatCompletion(t, w, sampleAnalysisJSON)
	})

	analysis, err := client.Analyze(context.Background(), testPacket(nil))
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Summary != "A short summary" {
		t.Fatalf("unexpected summary: %q", analysis.Summary)
	}

	req := decodePostedChat(t, posted)
	if req.Model != "google/diffusiongemma-26b-a4b-it" {
		t.Fatalf("unexpected model: %q", req.Model)
	}
	raw := userContent(t, req)
	var content string
	if err := json.Unmarshal(raw, &content); err != nil {
		t.Fatalf("text-only user content should remain a string: %v\n%s", err, raw)
	}
	if !strings.Contains(content, "Hello world") {
		t.Fatalf("user text missing markdown:\n%s", content)
	}
	if strings.Contains(string(posted), "image_url") {
		t.Fatalf("text-only request included image_url:\n%s", posted)
	}
}

func TestAnalyzeIncludesImageURLForHTTPSCover(t *testing.T) {
	const cover = "https://cdn.example.com/cover.jpg"
	var posted []byte
	requests := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		var err error
		posted, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		writeChatCompletion(t, w, sampleAnalysisJSON)
	})

	analysis, err := client.Analyze(context.Background(), testPacket(cover))
	if err != nil {
		t.Fatal(err)
	}
	if !analysis.Provenance.ImageSent {
		t.Fatal("expected imageSent provenance")
	}
	if analysis.Provenance.ImageSkipped != "" {
		t.Fatalf("unexpected imageSkipped: %q", analysis.Provenance.ImageSkipped)
	}

	req := decodePostedChat(t, posted)
	raw := userContent(t, req)
	var parts []contentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		t.Fatalf("multimodal user content should be a part array: %v\n%s", err, raw)
	}
	var sawText, sawImage bool
	for _, part := range parts {
		switch part.Type {
		case "text":
			sawText = true
			if !strings.Contains(part.Text, "Hello world") {
				t.Fatalf("text part missing markdown:\n%s", part.Text)
			}
		case "image_url":
			sawImage = true
			if part.ImageURL == nil || part.ImageURL.URL != cover {
				t.Fatalf("unexpected image_url part: %+v", part.ImageURL)
			}
		}
	}
	if !sawText || !sawImage {
		t.Fatalf("expected text and image_url parts, got %+v", parts)
	}
	if requests != 1 {
		t.Fatalf("expected one provider request, got %d", requests)
	}
}

func TestAnalyzeSkipsUnusableCoverImages(t *testing.T) {
	cases := []struct {
		name  string
		image any
	}{
		{name: "missing"},
		{name: "empty", image: ""},
		{name: "whitespace", image: "   "},
		{name: "data-uri", image: "data:image/png;base64,AAAA"},
		{name: "file", image: "file:///tmp/cover.png"},
		{name: "blob", image: "blob:https://example.com/cover"},
		{name: "relative", image: "/images/cover.jpg"},
		{name: "no-host", image: "https://"},
		{name: "non-string", image: 42},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var posted []byte
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				var err error
				posted, err = io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				writeChatCompletion(t, w, sampleAnalysisJSON)
			})

			var image any
			if tc.name != "missing" {
				image = tc.image
			}
			analysis, err := client.Analyze(context.Background(), testPacket(image))
			if err != nil {
				t.Fatal(err)
			}
			if analysis.Provenance.ImageSent {
				t.Fatal("unusable image should not be marked sent")
			}
			if strings.Contains(string(posted), "image_url") {
				t.Fatalf("unusable image included image_url:\n%s", posted)
			}
			var content string
			if err := json.Unmarshal(userContent(t, decodePostedChat(t, posted)), &content); err != nil {
				t.Fatalf("expected string user content: %v", err)
			}
		})
	}
}

func TestAnalyzeProvenanceFromConfigOmitsAPIKey(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test-secret-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		writeChatCompletion(t, w, sampleAnalysisJSON)
	})

	analysis, err := client.Analyze(context.Background(), testPacket("https://cdn.example.com/cover.jpg"))
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Provenance.Model != "google/diffusiongemma-26b-a4b-it" {
		t.Fatalf("unexpected model: %q", analysis.Provenance.Model)
	}
	if analysis.Provenance.ProviderHost != parsed.Host {
		t.Fatalf("unexpected provider host: %q want %q", analysis.Provenance.ProviderHost, parsed.Host)
	}
	if analysis.Provenance.PromptVersion != PromptVersion {
		t.Fatalf("unexpected prompt version: %q", analysis.Provenance.PromptVersion)
	}
	if !analysis.Provenance.ImageSent {
		t.Fatal("expected imageSent")
	}
	if analysis.Provenance.AnalyzedAt != "2026-08-13T12:00:00Z" {
		t.Fatalf("unexpected analyzedAt: %q", analysis.Provenance.AnalyzedAt)
	}

	raw, err := json.Marshal(analysis)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-test-secret-key") || strings.Contains(string(raw), "Bearer") {
		t.Fatalf("analysis leaked API key material:\n%s", raw)
	}
}

func TestAnalyzeDecodesFencedJSON(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeChatCompletion(t, w, "```json\n"+sampleAnalysisJSON+"\n```")
	})

	analysis, err := client.Analyze(context.Background(), testPacket(nil))
	if err != nil {
		t.Fatal(err)
	}
	if analysis.PageType != "article" || analysis.Summary != "A short summary" {
		t.Fatalf("fenced JSON did not decode: %+v", analysis)
	}
}

func TestAnalyzeDecodesJSONWrappedInProse(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeChatCompletion(t, w, "Here is the analysis:\n"+sampleAnalysisJSON+"\nHope this helps.")
	})

	analysis, err := client.Analyze(context.Background(), testPacket(nil))
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Summary != "A short summary" {
		t.Fatalf("wrapped JSON did not decode: %+v", analysis)
	}
}

func TestAnalyzeChunkedPathSendsImageOnlyOnContent(t *testing.T) {
	var posted [][]byte
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		posted = append(posted, body)
		writeChatCompletion(t, w, sampleAnalysisJSON)
	})
	client.chunkChars = 40

	packet := testPacket("https://cdn.example.com/cover.jpg")
	packet.Content.Markdown = strings.Repeat("paragraph one\n\n", 8)
	if _, err := client.Analyze(context.Background(), packet); err != nil {
		t.Fatal(err)
	}
	if len(posted) < 3 {
		t.Fatalf("expected chunk requests plus synthesis, got %d", len(posted))
	}
	for i, body := range posted[:len(posted)-1] {
		if !strings.Contains(string(body), "image_url") {
			t.Fatalf("content chunk %d missing image_url:\n%s", i+1, body)
		}
	}
	if strings.Contains(string(posted[len(posted)-1]), "image_url") {
		t.Fatalf("synthesis request should stay text-only:\n%s", posted[len(posted)-1])
	}
}

func TestUsableCoverImageURL(t *testing.T) {
	if got := usableCoverImageURL(testPacket("https://cdn.example.com/a.png")); got != "https://cdn.example.com/a.png" {
		t.Fatalf("https URL rejected: %q", got)
	}
	if got := usableCoverImageURL(testPacket("http://example.com/a.png")); got != "http://example.com/a.png" {
		t.Fatalf("http URL rejected: %q", got)
	}
	if got := usableCoverImageURL(testPacket("data:image/png;base64,xx")); got != "" {
		t.Fatalf("data URI accepted: %q", got)
	}
}

func TestDecodeAnalysisJSON(t *testing.T) {
	analysis, err := decodeAnalysisJSON("```json\n" + sampleAnalysisJSON + "\n```")
	if err != nil {
		t.Fatal(err)
	}
	if analysis.PageType != "article" {
		t.Fatalf("unexpected pageType: %q", analysis.PageType)
	}

	analysis, err = decodeAnalysisJSON("prefix " + sampleAnalysisJSON + " suffix")
	if err != nil {
		t.Fatal(err)
	}
	if analysis.Summary != "A short summary" {
		t.Fatalf("extracted object mismatch: %+v", analysis)
	}
}

func TestProviderHost(t *testing.T) {
	if got := providerHost("https://integrate.api.nvidia.com/v1"); got != "integrate.api.nvidia.com" {
		t.Fatalf("unexpected host: %q", got)
	}
	if got := providerHost("http://127.0.0.1:8000/v1"); got != "127.0.0.1:8000" {
		t.Fatalf("unexpected local host: %q", got)
	}
}

func writeProviderError(t *testing.T, w http.ResponseWriter, status int, message string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    "BadRequestError",
			"code":    status,
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestAnalyzeRetriesWithoutImageWhenProviderCannotFetchCover(t *testing.T) {
	const cover = "https://upload.wikimedia.org/wikipedia/commons/cover.jpg"
	cases := []struct {
		name    string
		message string
	}{
		{
			name:    "upstream-403",
			message: "Failed to fetch media from URL: upstream returned HTTP 403.",
		},
		{
			name:    "upstream-401",
			message: "Failed to fetch media from URL: upstream returned HTTP 401.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var posted [][]byte
			client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				posted = append(posted, body)
				if len(posted) == 1 {
					writeProviderError(t, w, http.StatusBadRequest, tc.message)
					return
				}
				writeChatCompletion(t, w, sampleAnalysisJSON)
			})

			analysis, err := client.Analyze(context.Background(), testPacket(cover))
			if err != nil {
				t.Fatal(err)
			}
			if analysis.Summary != "A short summary" {
				t.Fatalf("unexpected summary after retry: %q", analysis.Summary)
			}
			if analysis.Provenance.ImageSent {
				t.Fatal("text-only retry must not report imageSent")
			}
			if analysis.Provenance.ImageSkipped != ImageSkippedProviderMediaFetch {
				t.Fatalf("unexpected imageSkipped: %q", analysis.Provenance.ImageSkipped)
			}
			if analysis.Provenance.Model != "google/diffusiongemma-26b-a4b-it" {
				t.Fatalf("unexpected model: %q", analysis.Provenance.Model)
			}
			parsed, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			if analysis.Provenance.ProviderHost != parsed.Host {
				t.Fatalf("unexpected provider host: %q", analysis.Provenance.ProviderHost)
			}
			if analysis.Provenance.PromptVersion != PromptVersion {
				t.Fatalf("unexpected prompt version: %q", analysis.Provenance.PromptVersion)
			}

			if len(posted) != 2 {
				t.Fatalf("expected one retry, got %d requests", len(posted))
			}
			if !strings.Contains(string(posted[0]), "image_url") {
				t.Fatalf("first request should include image_url:\n%s", posted[0])
			}
			if strings.Contains(string(posted[1]), "image_url") {
				t.Fatalf("retry must omit image_url:\n%s", posted[1])
			}
			var content string
			if err := json.Unmarshal(userContent(t, decodePostedChat(t, posted[1])), &content); err != nil {
				t.Fatalf("retry user content should be a string: %v\n%s", err, posted[1])
			}
			if !strings.Contains(content, "Hello world") {
				t.Fatalf("retry dropped page text:\n%s", content)
			}

			raw, err := json.Marshal(analysis)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "sk-test-secret-key") || strings.Contains(string(raw), "Bearer") {
				t.Fatalf("analysis leaked API key material:\n%s", raw)
			}
		})
	}
}

func TestAnalyzeDoesNotRetryOnProvider500(t *testing.T) {
	requests := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "image_url") {
			t.Fatalf("500 path should still send image_url on the only request:\n%s", body)
		}
		writeProviderError(t, w, http.StatusInternalServerError, "Failed to fetch media from URL: upstream returned HTTP 403.")
	})

	_, err := client.Analyze(context.Background(), testPacket("https://cdn.example.com/cover.jpg"))
	if err == nil {
		t.Fatal("expected provider 500 to fail analysis")
	}
	if !strings.Contains(err.Error(), "provider HTTP 500") {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("500 must not retry, got %d requests", requests)
	}
}

func TestAnalyzeDoesNotRetryTextOnlyMediaFetchError(t *testing.T) {
	requests := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "image_url") {
			t.Fatalf("text-only packet included image_url:\n%s", body)
		}
		writeProviderError(t, w, http.StatusBadRequest, "Failed to fetch media from URL: upstream returned HTTP 403.")
	})

	_, err := client.Analyze(context.Background(), testPacket(nil))
	if err == nil {
		t.Fatal("expected text-only media-fetch error to fail")
	}
	if !strings.Contains(err.Error(), "provider HTTP 400") {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("text-only media-fetch error must not retry, got %d requests", requests)
	}
}

func TestAnalyzeDoesNotRetryOnProvider401Auth(t *testing.T) {
	requests := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeProviderError(t, w, http.StatusUnauthorized, "Failed to fetch media from URL: upstream returned HTTP 401.")
	})

	_, err := client.Analyze(context.Background(), testPacket("https://cdn.example.com/cover.jpg"))
	if err == nil {
		t.Fatal("expected provider 401 to fail analysis")
	}
	if !strings.Contains(err.Error(), "provider HTTP 401") {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("401 auth must not retry, got %d requests", requests)
	}
}

func TestAnalyzeDoesNotRetryOnJSONDecodeError(t *testing.T) {
	requests := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"choices":[{"message":{"content":"not-json"}}]}`); err != nil {
			t.Fatal(err)
		}
	})

	_, err := client.Analyze(context.Background(), testPacket("https://cdn.example.com/cover.jpg"))
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
	if !strings.Contains(err.Error(), "decode analysis JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("JSON decode error must not retry, got %d requests", requests)
	}
}

func TestAnalyzeDoesNotRetryOnUnrelated400(t *testing.T) {
	requests := 0
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeProviderError(t, w, http.StatusBadRequest, "invalid model")
	})

	_, err := client.Analyze(context.Background(), testPacket("https://cdn.example.com/cover.jpg"))
	if err == nil {
		t.Fatal("expected unrelated 400 to fail analysis")
	}
	if !strings.Contains(err.Error(), "provider HTTP 400") {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("unrelated 400 must not retry, got %d requests", requests)
	}
}

func TestIsProviderMediaFetchError(t *testing.T) {
	nim403 := &providerHTTPError{
		StatusCode: http.StatusBadRequest,
		Body:       `{"error":{"message":"Failed to fetch media from URL: upstream returned HTTP 403.","type":"BadRequestError","code":400}}`,
	}
	if !isProviderMediaFetchError(nim403) {
		t.Fatal("expected NVIDIA 400 media-fetch body to match")
	}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "400-fetch-media", err: &providerHTTPError{StatusCode: 400, Body: "could not fetch media"}, want: true},
		{name: "400-failed-media", err: &providerHTTPError{StatusCode: 400, Body: "Failed media download"}, want: true},
		{name: "422-upstream-401", err: &providerHTTPError{StatusCode: 422, Body: "upstream returned HTTP 401"}, want: true},
		{name: "401-auth", err: &providerHTTPError{StatusCode: 401, Body: "Failed to fetch media from URL: upstream returned HTTP 401."}, want: false},
		{name: "500", err: &providerHTTPError{StatusCode: 500, Body: "Failed to fetch media from URL: upstream returned HTTP 403."}, want: false},
		{name: "400-unrelated", err: &providerHTTPError{StatusCode: 400, Body: "invalid model"}, want: false},
		{name: "timeout", err: errors.New("context deadline exceeded"), want: false},
		{name: "decode", err: fmt.Errorf("decode provider response: %w", errors.New("invalid character")), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isProviderMediaFetchError(tc.err); got != tc.want {
				t.Fatalf("isProviderMediaFetchError(%v)=%v want %v", tc.err, got, tc.want)
			}
		})
	}
}
