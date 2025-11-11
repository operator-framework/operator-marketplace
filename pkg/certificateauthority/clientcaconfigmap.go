package certificateauthority

import (
	"crypto/x509"
	"sync"
)

type ClientCAStore struct {
	clientCA *x509.CertPool
	mutex sync.RWMutex
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
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.clientCA.AppendCertsFromPEM(newCAPEM)
}

func (c *ClientCAStore) GetCA() *x509.CertPool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.clientCA
}