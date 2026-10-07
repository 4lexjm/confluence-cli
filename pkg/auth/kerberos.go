package auth

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/spnego"
)

// KerberosTransport is an http.RoundTripper that negotiates Kerberos/SPNEGO
// authentication against a Confluence Data Center instance.
type KerberosTransport struct {
	inner      http.RoundTripper
	krb5Client *client.Client
}

// KerberosExpiredError indicates ticket expiration.
type KerberosExpiredError struct {
	cause error
}

func (e *KerberosExpiredError) Error() string {
	return fmt.Sprintf("Kerberos ticket expired or invalid: %v\nHint: run `kinit` to obtain a fresh ticket", e.cause)
}

// NewKerberosTransport creates a KerberosTransport.
func NewKerberosTransport(inner http.RoundTripper) (*KerberosTransport, error) {
	if inner == nil {
		inner = http.DefaultTransport
	}

	confPath := os.Getenv("KRB5_CONFIG")
	if confPath == "" {
		confPath = "/etc/krb5.conf"
	}

	krb5Conf, err := config.Load(confPath)
	if err != nil {
		return nil, fmt.Errorf("load krb5 config (%s): %w\nHint: ensure krb5.conf exists", confPath, err)
	}

	ccachePath := os.Getenv("KRB5CCNAME")
	ccache, err := credentials.LoadCCache(ccachePath)
	if err != nil {
		return nil, fmt.Errorf("load Kerberos credential cache: %w\nHint: run `kinit <user>@<REALM>` to obtain a ticket", err)
	}

	krb5Client, err := client.NewFromCCache(ccache, krb5Conf, client.DisablePAFXFAST(true))
	if err != nil {
		return nil, fmt.Errorf("create Kerberos client from ccache: %w", err)
	}

	return &KerberosTransport{inner: inner, krb5Client: krb5Client}, nil
}

// RoundTrip executes the SPNEGO negotiated request.
func (t *KerberosTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	httpClient := &http.Client{Transport: t.inner}
	spnegoClient := spnego.NewClient(t.krb5Client, httpClient, "")

	clone := req.Clone(req.Context())
	resp, err := spnegoClient.Do(clone)
	if err != nil {
		return nil, &KerberosExpiredError{cause: err}
	}
	return resp, nil
}

// ErrKerberosUnavailable indicates Kerberos ccache or config is missing.
var ErrKerberosUnavailable = errors.New("kerberos credentials unavailable")
