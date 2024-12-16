package whois

import "testing"

func Test_getDomainFromURL(t *testing.T) {
	testCases := []struct {
		name     string
		inputURL string
		want     string
		wantErr  bool
	}{
		{
			name:     "Normal HTTP URL",
			inputURL: "http://www.google.com",
			want:     "google.com",
			wantErr:  false,
		},
		{
			name:     "Normal HTTPS URL",
			inputURL: "https://www.example.com",
			want:     "example.com",
			wantErr:  false,
		},
		{
			name:     "URL without protocol",
			inputURL: "www.bing.com",
			want:     "bing.com",
			wantErr:  false,
		},
		{
			name:     "Invalid URL",
			inputURL: "invalid_url",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "Subdomain with TLD+1",
			inputURL: "https://subdomain.google.com",
			want:     "google.com",
			wantErr:  false,
		},
		{
			name:     "Subdomain without TLD+1",
			inputURL: "https://www.google.co.uk",
			want:     "google.co.uk",
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getDomainFromURL(tc.inputURL)
			if (err != nil) != tc.wantErr {
				t.Errorf("getDomainFromURL(%q) error = `%v`, wantErr `%v`", tc.inputURL, err, tc.wantErr)
				return
			}

			if got != tc.want {
				t.Errorf("getDomainFromURL(%q) = %v, want %v", tc.inputURL, got, tc.want)
			}
		})
	}
}
