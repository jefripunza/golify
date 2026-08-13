package main

import (
	"testing"
)

func TestNormalizeDomainSuffixValidation(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		// valid — root domains only (Pak Jefri's rule)
		{"example.com", false},
		{"simtaru.online", false},
		{"wajadi.online", false},
		{"foo.co.id", false},
		{"my-site.dev", false},
		{"example.xn--p1ai", false}, // punycode TLD (рф) — valid TLD
		// www ≡ apex — normalized to the same bare host
		{"www.example.com", false},
		{"WWW.SIMTARU.ONLINE", false},
		{"https://www.sawang.tech/", false},
		// invalid — subdomains are NOT allowed (hard rule)
		{"sub.example.com", true},  // subdomain rejected
		{"a.b.co.id", true},        // subdomain of a co.id root rejected
		{"www.sub.example.com", true}, // www + subdomain rejected
		{"golify.sawang.tech", true},  // subdomain rejected
		{"test.simtaru.online", true}, // subdomain rejected
		// invalid — TLD / format
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
