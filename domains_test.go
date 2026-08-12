package main

import (
	"testing"
)

func TestNormalizeDomainSuffixValidation(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		// valid
		{"example.com", false},
		{"simtaru.online", false},
		{"wajadi.online", false},
		{"foo.co.id", false},
		{"sub.example.com", false},
		{"a.b.co.id", false},
		{"example.xn--p1ai", false}, // punycode TLD (рф) — valid TLD
		{"my-site.dev", false},
		// www ≡ apex — normalized to the same bare host
		{"www.example.com", false},
		{"WWW.SIMTARU.ONLINE", false},
		{"https://www.sawang.tech/", false},
		{"www.sub.example.com", false},
		// invalid
		{"noservice.example", true},  // .example not a real TLD
		{"foo.badexample", true},     // .badexample not a TLD
		{"example.invalidtld", true}, // .invalidtld not a TLD
		{"justonelonglabel", true},   // no dot
		{"-bad.com", true},           // leading dash
		{"bad-.com", true},           // trailing dash
		{"co.id", true},              // bare public suffix — not a domain
		{"www", true},                // "www" alone is not a domain
		{"www.example", true},        // .example not a TLD
	}
	for _, tc := range cases {
		_, err := normalizeDomain(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("normalizeDomain(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
		}
	}
}
