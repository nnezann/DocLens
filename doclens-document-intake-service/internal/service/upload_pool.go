package service

import (
	"fmt"
	"sync"
	"time"
)

type UploadPoolConfig struct {
	Workers          int
	TenantRate       float64
	TenantBurst      int
	FailureThreshold int
	OpenDuration     time.Duration
}

type uploadPool struct {
	sem       chan struct{}
	config    UploadPoolConfig
	mu        sync.Mutex
	tenants   map[string]*tenantBucket
	failures  int
	openUntil time.Time
}

type tenantBucket struct {
	tokens float64
	last   time.Time
}

func newUploadPool(config UploadPoolConfig) (*uploadPool, error) {
	if config.Workers <= 0 || config.TenantRate <= 0 || config.TenantBurst <= 0 ||
		config.FailureThreshold <= 0 || config.OpenDuration <= 0 {
		return nil, fmt.Errorf("upload pool configuration values must be positive")
	}
	return &uploadPool{
		sem: make(chan struct{}, config.Workers), config: config, tenants: make(map[string]*tenantBucket),
	}, nil
}

func (p *uploadPool) acquire(organizationID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if now.Before(p.openUntil) {
		return fmt.Errorf("object storage circuit is open")
	}
	bucket := p.tenants[organizationID]
	if bucket == nil {
		bucket = &tenantBucket{tokens: float64(p.config.TenantBurst), last: now}
		p.tenants[organizationID] = bucket
	}
	bucket.tokens = minFloat(float64(p.config.TenantBurst), bucket.tokens+now.Sub(bucket.last).Seconds()*p.config.TenantRate)
	bucket.last = now
	if bucket.tokens < 1 {
		return fmt.Errorf("organization upload rate limit exceeded")
	}
	select {
	case p.sem <- struct{}{}:
		bucket.tokens--
		return nil
	default:
		return fmt.Errorf("upload pool is saturated")
	}
}

func (p *uploadPool) release() {
	<-p.sem
}

func (p *uploadPool) success() {
	p.mu.Lock()
	p.failures = 0
	p.mu.Unlock()
}

func (p *uploadPool) failure() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures++
	if p.failures >= p.config.FailureThreshold {
		p.openUntil = time.Now().Add(p.config.OpenDuration)
		p.failures = 0
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
