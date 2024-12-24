package main

import (
	"testing"
)

func Test_extractEmailsFromHTML(t *testing.T) {
	testCases := []struct {
		name        string
		domain      string
		htmlContent string
		expected    []string
	}{
		{
			name:        "Normal case",
			domain:      "example.com",
			htmlContent: `<p>Contact us at support@example.com or sales@example.com</p>`,
			expected:    []string{"support@example.com", "sales@example.com"},
		},
		{
			name:        "Emails in attributes",
			domain:      "example.com",
			htmlContent: `<a href="mailto:help@example.com">Help</a>`,
			expected:    []string{"help@example.com"},
		},
		{
			name:        "Edge case - no domain match",
			domain:      "example.com",
			htmlContent: `<p>Contact us at support@otherexample.com</p>`,
			expected:    []string{},
		},
		{
			name:        "Invalid input - malformed email",
			domain:      "example.com",
			htmlContent: `<p>Contact us at support@@otherexample.com</p>`,
			expected:    []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			emailRegex := buildEmailRegexp(tc.domain)
			emails := emailRegex.FindAllString(tc.htmlContent, -1)
			if len(emails) != len(tc.expected) {
				t.Errorf("expected %d emails, god %d", len(tc.expected), len(emails))
			}
			for i, email := range emails {
				if email != tc.expected[i] {
					t.Errorf("expected email %s, got %s", tc.expected[i], email)
				}
			}
		})

	}
}

func Test_createHTTPRequest(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		url             string
		headers         map[string]string
		expectedErr     bool
		expectedMethod  string
		expectedURL     string
		expectedHeaders map[string]string
	}{
		{
			name:            "Valid GET Request with Headers",
			method:          "GET",
			url:             "https://example.com",
			headers:         map[string]string{"Authorization": "Bearer token123", "Content-Type": "application/json"},
			expectedErr:     false,
			expectedMethod:  "GET",
			expectedURL:     "https://example.com",
			expectedHeaders: map[string]string{"Authorization": "Bearer token123", "Content-Type": "application/json"},
		},
		{
			name:            "Valid POST Request without Headers",
			method:          "POST",
			url:             "https://example.com/api",
			headers:         map[string]string{},
			expectedErr:     false,
			expectedMethod:  "POST",
			expectedURL:     "https://example.com/api",
			expectedHeaders: map[string]string{},
		},
		{
			name:            "Invalid Empty URL",
			method:          "GET",
			url:             "",
			headers:         nil,
			expectedErr:     true,
			expectedMethod:  "GET",
			expectedURL:     "",
			expectedHeaders: map[string]string{},
		},
		{
			name:            "Invalid HTTP Method",
			method:          "INVALID",
			url:             "",
			headers:         nil,
			expectedErr:     true,
			expectedMethod:  "INVALID",
			expectedURL:     "",
			expectedHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := createHTTPRequest(tt.method, tt.url, tt.headers)

			// Check error
			if (err != nil) != tt.expectedErr {
				t.Errorf("Expected error: %v, got %v", tt.expectedErr, err)
			}

			if err != nil {
				return
			}

			if req.Method != tt.expectedMethod {
				t.Errorf("Expected method: %s, got %s", req.Method, tt.expectedMethod)
			}

			for key, expectedValue := range tt.expectedHeaders {
				if gotValue := req.Header.Get(key); gotValue != expectedValue {
					t.Errorf("Expected header %s: %s, got: %s", key, expectedValue, gotValue)
				}
			}

			// Check unexpected headers
			for key := range req.Header {
				if _, exists := tt.expectedHeaders[key]; !exists {
					t.Errorf("Unexpected header set: %s", key)
				}
			}
		})
	}
}
