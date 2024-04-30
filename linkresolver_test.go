package main

import "testing"

func TestParseLinkAndAnchor(t *testing.T) {
	tests := []struct {
		link       string
		wantAnchor string
		wantLink   string
	}{
		{link: "https://example.com", wantAnchor: "", wantLink: "https://example.com"},
		{link: "index.md", wantAnchor: "", wantLink: "index.md"},
		{link: "index.md#foo", wantAnchor: "foo", wantLink: "index.md"},
		{link: "bar/index.md#foo", wantAnchor: "foo", wantLink: "bar/index.md"},
		{link: "#foo", wantAnchor: "foo", wantLink: ""},
	}

	for _, tt := range tests {
		t.Run(tt.link, func(t *testing.T) {
			gotLink, gotAnchor, err := parseLinkAndAnchor(tt.link)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if gotAnchor != tt.wantAnchor {
				t.Errorf("got anchor %q, want %q", gotAnchor, tt.wantAnchor)
			}
			if gotLink != tt.wantLink {
				t.Errorf("got link %q, want %q", gotLink, tt.wantLink)
			}
		})
	}
}
