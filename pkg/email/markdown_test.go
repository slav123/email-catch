package email

import "testing"

func TestMarkdownConverter_getAttachmentURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		baseURL    string
		folderPath string
		filename   string
		want       string
	}{
		{
			name:       "known folder komunikacja-pro",
			baseURL:    "https://example.com",
			folderPath: "komunikacja-pro/a/b",
			filename:   "file.png",
			want:       "https://img.komunikacja.pro/emails/komunikacja-pro/a/b/file.png",
		},
		{
			name:       "real folderPath",
			baseURL:    "https://example.com",
			folderPath: "komunikacja-pro/2025/12/20251211_191055.000000_3c6a6f5f",
			filename:   "image001.png",
			want:       "https://img.komunikacja.pro/emails/komunikacja-pro/2025/12/20251211_191055.000000_3c6a6f5f/image001.png",
		},
		{
			name:       "known folder faktury-hib",
			baseURL:    "https://example.com",
			folderPath: "faktury-hib/2025/12",
			filename:   "invoice.pdf",
			want:       "https://img.hib.pl/emails/faktury-hib/2025/12/invoice.pdf",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := NewMarkdownConverter(tt.baseURL, tt.folderPath)
			got := mc.getAttachmentURL(tt.filename)
			t.Logf("getAttachmentURL(%q) baseURL=%q folderPath=%q => %q", tt.filename, tt.baseURL, tt.folderPath, got)
			if got != tt.want {
				t.Fatalf("unexpected url: got=%q want=%q", got, tt.want)
			}
		})
	}
}
