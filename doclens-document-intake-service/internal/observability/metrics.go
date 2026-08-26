package observability

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

type IntakeMetrics struct {
	activeWorkers       atomic.Int64
	queueDepth          atomic.Int64
	rejectedTotal       atomic.Uint64
	bytesStreamedTotal  atomic.Uint64
	presignedURLIssued  atomic.Uint64
	confirmLatencyNanos atomic.Uint64
	confirmations       atomic.Uint64
}

func NewIntakeMetrics() *IntakeMetrics {
	return &IntakeMetrics{}
}

func (m *IntakeMetrics) WorkerStarted()            { m.activeWorkers.Add(1) }
func (m *IntakeMetrics) WorkerFinished()           { m.activeWorkers.Add(-1) }
func (m *IntakeMetrics) SetQueueDepth(depth int64) { m.queueDepth.Store(depth) }
func (m *IntakeMetrics) UploadRejected()           { m.rejectedTotal.Add(1) }
func (m *IntakeMetrics) AddStreamedBytes(bytes int64) {
	if bytes > 0 {
		m.bytesStreamedTotal.Add(uint64(bytes))
	}
}
func (m *IntakeMetrics) PresignedURLIssued() { m.presignedURLIssued.Add(1) }
func (m *IntakeMetrics) ObserveConfirmation(duration time.Duration) {
	m.confirmLatencyNanos.Add(uint64(duration))
	m.confirmations.Add(1)
}

func (m *IntakeMetrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		confirmations := m.confirmations.Load()
		var latencyMS uint64
		if confirmations > 0 {
			latencyMS = m.confirmLatencyNanos.Load() / confirmations / uint64(time.Millisecond)
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(w,
			"intake_pool_active_workers %d\nintake_pool_queue_depth %d\nintake_pool_rejected_total %d\nintake_upload_bytes_streamed_total %d\nintake_presigned_url_issued_total %d\nintake_upload_confirm_latency_ms %d\n",
			m.activeWorkers.Load(), m.queueDepth.Load(), m.rejectedTotal.Load(),
			m.bytesStreamedTotal.Load(), m.presignedURLIssued.Load(), latencyMS)
	})
}
