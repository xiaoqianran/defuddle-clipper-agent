package protocol

import "testing"

func validPacket() ContentPacket {
	return ContentPacket{
		ProtocolVersion: Version,
		CaptureID:       "123e4567-e89b-12d3-a456-426614174000",
		CapturedAt:      "2026-08-13T10:00:00Z",
		Source: Source{
			URL:   "https://example.com/article",
			Title: "Example",
		},
		Content: Content{Markdown: "# Hello"},
	}
}

func TestValidate(t *testing.T) {
	p := validPacket()
	if err := p.Validate(); err != nil {
		t.Fatalf("valid packet rejected: %v", err)
	}
}

func TestValidateRejectsUnsafeCaptureID(t *testing.T) {
	p := validPacket()
	p.CaptureID = "../../escape"
	if err := p.Validate(); err == nil {
		t.Fatal("expected unsafe capture id to fail")
	}
}

func TestValidateRejectsNonHTTPURL(t *testing.T) {
	p := validPacket()
	p.Source.URL = "file:///etc/passwd"
	if err := p.Validate(); err == nil {
		t.Fatal("expected non-http URL to fail")
	}
}
