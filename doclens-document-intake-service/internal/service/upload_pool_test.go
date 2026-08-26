package service

import (
	"testing"
	"time"
)

func TestUploadPoolRejectsSaturationAndIsolatesTenants(t *testing.T) {
	pool, err := newUploadPool(UploadPoolConfig{
		Workers: 1, TenantRate: 100, TenantBurst: 2, FailureThreshold: 2, OpenDuration: time.Minute,
	})
	if err != nil {
		t.Fatalf("new upload pool: %v", err)
	}

	if err := pool.acquire("org_1"); err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if err := pool.acquire("org_1"); err == nil {
		t.Fatal("expected saturated pool to reject the second upload")
	}
	pool.release()
	if err := pool.acquire("org_2"); err != nil {
		t.Fatalf("expected another tenant to acquire after capacity was released: %v", err)
	}
	pool.release()
}

func TestUploadPoolRateLimitIsPerTenant(t *testing.T) {
	pool, err := newUploadPool(UploadPoolConfig{
		Workers: 2, TenantRate: 0.001, TenantBurst: 1, FailureThreshold: 2, OpenDuration: time.Minute,
	})
	if err != nil {
		t.Fatalf("new upload pool: %v", err)
	}
	if err := pool.acquire("org_1"); err != nil {
		t.Fatalf("org_1 acquire: %v", err)
	}
	pool.release()
	if err := pool.acquire("org_1"); err == nil {
		t.Fatal("expected org_1 rate limit to reject")
	}
	if err := pool.acquire("org_2"); err != nil {
		t.Fatalf("expected org_2 to have an independent bucket: %v", err)
	}
	pool.release()
}
