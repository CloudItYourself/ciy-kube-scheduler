package httpserver

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"
)

type CertReloader struct {
	CertFile          string // path to the x509 certificate for https
	KeyFile           string // path to the x509 private key matching `CertFile`
	CachedCert        *tls.Certificate
	CachedCertModTime time.Time
	CachedKeyModTime  time.Time
}

// Implementation for tls.Config.GetCertificate useful when using
// Kubernetes Secrets which update the filesystem at runtime.
func (cr *CertReloader) GetCertificate(h *tls.ClientHelloInfo) (*tls.Certificate, error) {
	key_stat, key_err := os.Stat(cr.KeyFile)
	cert_stat, cert_err := os.Stat(cr.CertFile)
	if key_err != nil || cert_err != nil {
		if cr.CachedCert != nil {
			log.Printf("Failed to get stat details for cert file... returning previous cert")
			return cr.CachedCert, nil
		}
		return nil, fmt.Errorf("failed loading certificates: key_error: %w, cert_error: %w", key_err, cert_err)
	}

	if cr.CachedCert == nil || (key_stat.ModTime().After(cr.CachedKeyModTime) && cert_stat.ModTime().After(cr.CachedCertModTime)) {
		pair, err := tls.LoadX509KeyPair(cr.CertFile, cr.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed loading tls key pair: %w", err)
		}

		cr.CachedCert = &pair
		cr.CachedCertModTime = cert_stat.ModTime()
		cr.CachedKeyModTime = key_stat.ModTime()
	}

	return cr.CachedCert, nil
}
