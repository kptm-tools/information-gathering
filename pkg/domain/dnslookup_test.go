package domain

import "testing"

func Test_HasDNSKeyRecord(t *testing.T) {
	var testCases = []struct {
		name     string
		input    []DNSRecord
		expected bool
	}{
		{
			name: "DNSRecordArray that contains DNSKey record",
			input: []DNSRecord{
				{
					Type: DNSKeyRecord,
					Name: "example.com",
					TTL:  5,
				},
				{
					Type:  ARecord,
					Name:  "example.com",
					TTL:   5,
					Value: "192.168.0.0",
				},
			},
			expected: true,
		},
		{
			name: "DNSRecordArray that doesn't contain DNSKey record",
			input: []DNSRecord{
				{
					Type: AAAARecord,
					Name: "example.com",
					TTL:  5,
				},
				{
					Type:  ARecord,
					Name:  "example.com",
					TTL:   5,
					Value: "192.168.0.0",
				},
			},
			expected: false,
		},
		{
			name:     "Empty DNSRecordArray",
			input:    []DNSRecord{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := HasDNSKeyRecord(tc.input)

			if res != tc.expected {
				t.Errorf("Incorrect result, expected `%v`, got `%v`", tc.expected, res)
			}
		})
	}
}
