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
