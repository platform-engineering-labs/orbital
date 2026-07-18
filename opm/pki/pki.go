package pki

type CertType string

const (
	CertCA     CertType = "CA"
	CertIssuer CertType = "Issuer"
	CertUser   CertType = "User"
	CertCRL    CertType = "CRL"
)
