package certificateauthority

import "crypto/x509"

type ClientCAStore struct {
	clientCA *x509.CertPool
}

func NewClientCAStore(certpool *x509.CertPool) *ClientCAStore {
	if certpool == nil {
		certpool = x509.NewCertPool()
	}
	return &ClientCAStore{clientCA: certpool}
}

func (c *ClientCAStore) Update(newCAPEM []byte) {
	if newCAPEM == nil {
		return
	}
	c.clientCA.AppendCertsFromPEM(newCAPEM)
}

func (c *ClientCAStore) GetCA() *x509.CertPool {
	return c.clientCA
}