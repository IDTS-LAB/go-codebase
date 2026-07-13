package prometheus

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Recorder struct {
	mu         sync.RWMutex
	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec
	gauges     map[string]*prometheus.GaugeVec
}

func NewRecorder() *Recorder {
	return &Recorder{
		counters:   make(map[string]*prometheus.CounterVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
	}
}

func (r *Recorder) IncrementCounter(_ context.Context, name string, labels ...string) {
	vec := r.getOrCreateCounter(name)
	if len(labels)%2 != 0 {
		labels = append(labels, "")
	}
	vec.WithLabelValues(labels...).Inc()
}

func (r *Recorder) ObserveHistogram(_ context.Context, name string, value float64, labels ...string) {
	vec := r.getOrCreateHistogram(name)
	if len(labels)%2 != 0 {
		labels = append(labels, "")
	}
	vec.WithLabelValues(labels...).Observe(value)
}

func (r *Recorder) SetGauge(_ context.Context, name string, value float64, labels ...string) {
	vec := r.getOrCreateGauge(name)
	if len(labels)%2 != 0 {
		labels = append(labels, "")
	}
	vec.WithLabelValues(labels...).Set(value)
}

func (r *Recorder) getOrCreateCounter(name string) *prometheus.CounterVec {
	r.mu.RLock()
	v, ok := r.counters[name]
	r.mu.RUnlock()
	if ok {
		return v
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok = r.counters[name]; ok {
		return v
	}

	v = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: name,
	}, []string{"method", "path", "status"})
	r.counters[name] = v
	return v
}

func (r *Recorder) getOrCreateHistogram(name string) *prometheus.HistogramVec {
	r.mu.RLock()
	v, ok := r.histograms[name]
	r.mu.RUnlock()
	if ok {
		return v
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok = r.histograms[name]; ok {
		return v
	}

	v = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    name,
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})
	r.histograms[name] = v
	return v
}

func (r *Recorder) getOrCreateGauge(name string) *prometheus.GaugeVec {
	r.mu.RLock()
	v, ok := r.gauges[name]
	r.mu.RUnlock()
	if ok {
		return v
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if v, ok = r.gauges[name]; ok {
		return v
	}

	v = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: name,
	}, []string{"method", "path", "status"})
	r.gauges[name] = v
	return v
}
