//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"strings"

	"easymail/pkg/dkim"
)

const privKey = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDNUXO+Qsl1tw+GjrqFajz0ERSEUs1FHSL/+udZRWn1Atw8gz0+
tcGqhWChBDeU9gY5sKLEAZnX3FjC/T/IbqeiSM68kS5vLkzRI84eiJrm3+IieUqI
IicsO+WYxQs+JgVx5XhpPjX4SQjHtwEC2xKkWnEv+VPgO1JWdooURcSC6QIDAQAB
AoGAM9exRgVPIS4L+Ynohu+AXJBDgfX2ZtEomUIdUGk6i+cg/RaWTFNQh2IOOBn8
ftxwTfjP4HYXBm5Y60NO66klIlzm6ci303IePmjaj8tXQiriaVA0j4hmW+xgnqQX
PubFzfnR2eWLSOGChrNFbd3YABC+qttqT6vT0KpFyLdn49ECQQD3zYCpgelb0EBo
gc5BVGkbArcknhPwO39coPqKM4csu6cgI489XpF7iMh77nBTIiy6dsDdRYXZM3bq
ELTv6K4/AkEA1BwsIZG51W5DRWaKeobykQIB6FqHLW+Zhedw7BnxS8OflYAcSWi4
uGhq0DPojmhsmUC8jUeLe79CllZNP3LU1wJBAIZcoCnI7g5Bcdr4nyxfJ4pkw4cQ
S4FT0XAZPR/YZrADo8/SWCWPdFTGSuaf17nL6vLD1zljK/skY5LwshrvUCMCQQDM
MY7ehj6DVFHYlt2LFSyhInCZscTencgK24KfGF5t1JZlwt34YaMqjAMACmi/55Fc
e7DIxW5nI/nDZrOY+EAjAkA3BHUx3PeXkXJnXjlh7nGZmk/v8tB5fiofAwfXNfL7
bz0ZrT2Caz995Dpjommh5aMpCJvUGsrYCG6/Pbha9NXl
-----END RSA PRIVATE KEY-----`

func main() {
	email := []byte("From: test@easymail.my\r\n" +
		"Date: Mon, 29 Jun 2026 22:59:57 +0800\r\n" +
		"Subject: Test\r\n" +
		"To: recipient@example.com\r\n" +
		"Message-ID: <abc123@easymail.my>\r\n" +
		"\r\n" +
		"Hello world\r\n")

	opts := dkim.NewSigOptions()
	opts.PrivateKey = []byte(privKey)
	opts.Domain = "easymail.my"
	opts.Selector = "mail"
	opts.Canonicalization = "relaxed/relaxed"
	opts.Algo = "rsa-sha256"
	opts.Headers = []string{"from", "date", "subject", "to", "message-id"}
	opts.AddSignatureTimestamp = false

	err := dkim.Sign(&email, opts)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	parts := strings.SplitN(string(email), "\r\n", 15)
	for i, p := range parts {
		if strings.HasPrefix(p, "b=") {
			fmt.Printf("Line %d: b=<truncated:%d chars>\n", i, len(p)-2)
		} else {
			fmt.Printf("Line %d: %s\n", i, p)
		}
	}
	fmt.Printf("\n--- Check required tags in full header ---\n")
	fullHeader := strings.Join(parts[:6], "\n")
	for _, tag := range []string{"v=", "a=", "b=", "bh=", "c=", "d=", "h=", "s=", "q="} {
		if strings.Contains(fullHeader, tag) {
			fmt.Printf("  FOUND: %s\n", tag)
		} else {
			fmt.Printf("  MISSING: %s\n", tag)
		}
	}
	fmt.Printf("\n--- Full DKIM-Signature header ---\n%s\n", strings.Join(parts[:6], "\r\n"))
}
