package extractors

import (
	"context"
	"net"

	"easymail/internal/infrastructure/easydns"
)

func dnsLookupPTR(ctx context.Context, ip string) ([]string, error) {
	return easydns.GetDefault().LookupAddr(ctx, ip)
}

func dnsLookupMX(ctx context.Context, domain string) ([]*net.MX, error) {
	return easydns.GetDefault().LookupMX(ctx, domain)
}

func dnsLookupIP(ctx context.Context, host string) ([]net.IPAddr, error) {
	return easydns.GetDefault().LookupIPAddr(ctx, host)
}

// dnsLookupTXT looks up TXT records for a domain.
func dnsLookupTXT(ctx context.Context, domain string) ([]string, error) {
	return easydns.GetDefault().LookupTXT(ctx, domain)
}
