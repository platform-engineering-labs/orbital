package security

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	mgdns "github.com/miekg/dns"
	"github.com/platform-engineering-labs/orbital/opm/pki"
)

var ErrDNSSECValidationFailed = errors.New("dnssec validation failed: AD bit not set")

var PemTPL = `
-----BEGIN CERTIFICATE-----
%s
-----END CERTIFICATE-----
	`

var DNS = dns{
	resolvers: []net.IP{
		net.ParseIP("1.1.1.1"),
		net.ParseIP("8.8.8.8"),
	},
}

const (
	CAPrefix      = "_ops_ca"
	IssuerPrefix  = "_ops_issuer"
	SubjectPrefix = "_ops_u"
)

type dns struct {
	resolvers []net.IP
}

type LookupRequest struct {
	SKI       string
	Publisher string
}

type LookupResult struct {
	CA     []byte
	Issuer []byte
	SKI    []byte
}

func (d dns) Lookup(request *LookupRequest) *LookupResult {
	var err error
	result := &LookupResult{}

	result.CA, err = d.LookupOPSCERT(pki.CertCA, request)
	if err != nil {
		return nil
	}

	result.Issuer, err = d.LookupOPSCERT(pki.CertIssuer, request)
	if err != nil {
		return nil
	}

	result.SKI, err = d.LookupOPSCERT(pki.CertUser, request)
	if err != nil {
		return nil
	}

	return result
}

func (d dns) LookupOPSCERT(ctype pki.CertType, request *LookupRequest) ([]byte, error) {
	var name string

	switch ctype {
	case pki.CertCA:
		name = fmt.Sprintf("%s.%s", CAPrefix, request.Publisher)
	case pki.CertIssuer:
		name = fmt.Sprintf("%s.%s", IssuerPrefix, request.Publisher)
	case pki.CertUser:
		name = fmt.Sprintf("%s_%s.%s", SubjectPrefix, request.SKI, request.Publisher)
	}

	c := &mgdns.Client{
		Net:     "udp",
		Timeout: 5 * time.Second,
	}

	m := new(mgdns.Msg)
	m.SetQuestion(mgdns.Fqdn(name), mgdns.TypeTXT)
	m.SetEdns0(4096, true)
	m.RecursionDesired = true

	// Ensure resolver has a port
	resolver := d.resolvers[rand.Intn(len(d.resolvers))].String()
	if _, _, err := net.SplitHostPort(resolver); err != nil {
		resolver = net.JoinHostPort(resolver, "53")
	}

	resp, _, err := c.Exchange(m, resolver)
	if err != nil {
		// Retry over TCP on truncation or network error
		c.Net = "tcp"
		resp, _, err = c.Exchange(m, resolver)
		if err != nil {
			return nil, fmt.Errorf("dns query failed: %w", err)
		}
	}

	if resp.Truncated {
		c.Net = "tcp"
		resp, _, err = c.Exchange(m, resolver)
		if err != nil {
			return nil, fmt.Errorf("tcp retry after truncation failed: %w", err)
		}
	}

	if resp.Rcode != mgdns.RcodeSuccess {
		return nil, fmt.Errorf("dns error: %s", mgdns.RcodeToString[resp.Rcode])
	}

	// AD bit: set by a validating resolver when the full DNSSEC chain
	// of trust has been verified (signatures, NSEC/NSEC3 for NXDOMAIN, etc.)
	if !resp.AuthenticatedData {
		return nil, ErrDNSSECValidationFailed
	}

	var records []string
	for _, rr := range resp.Answer {
		if txt, ok := rr.(*mgdns.TXT); ok {
			records = append(records, txt.Txt...)
		}
	}

	return []byte(fmt.Sprintf(PemTPL, strings.Join(records, "\n"))), nil
}
