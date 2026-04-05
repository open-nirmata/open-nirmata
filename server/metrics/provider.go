package metrics

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// MetricsProvider manages OpenTelemetry metrics with Prometheus exporter for Fiber integration
type MetricsProvider struct {
	meterProvider *sdkmetric.MeterProvider
	meter         metric.Meter
	promExporter  *prometheus.Exporter

	// HTTP API Metrics
	HTTPRequestsTotal    metric.Int64Counter
	HTTPRequestDuration  metric.Float64Histogram
	HTTPRequestsInFlight metric.Int64UpDownCounter

	// TEI Service Metrics
	EmbeddingRequestsTotal   metric.Int64Counter
	EmbeddingRequestDuration metric.Float64Histogram
	EmbeddingErrors          metric.Int64Counter
	GrpcConnectionStatus     metric.Int64Gauge
	HttpFallbackCount        metric.Int64Counter

	// Connection Metrics
	ConnectionReconnects   metric.Int64Counter
	ConnectionHealthChecks metric.Int64Counter

	// General Service Metrics
	ActiveConnections metric.Int64Gauge
	RequestQueueSize  metric.Int64Gauge
}

// NewMetricsProvider creates a new OpenTelemetry metrics provider with Prometheus exporter
func NewMetricsProvider(serviceName, serviceVersion string) (*MetricsProvider, error) {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
	)

	// Create Prometheus exporter
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExporter),
	)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Create meter
	meter := meterProvider.Meter(serviceName)

	mp := &MetricsProvider{
		meterProvider: meterProvider,
		meter:         meter,
		promExporter:  promExporter,
	}

	// Initialize metrics
	if err := mp.initializeMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	return mp, nil
}

// initializeMetrics creates all the metric instruments
func (mp *MetricsProvider) initializeMetrics() error {
	var err error

	// HTTP API Metrics
	mp.HTTPRequestsTotal, err = mp.meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create http_requests_total counter: %w", err)
	}

	mp.HTTPRequestDuration, err = mp.meter.Float64Histogram(
		"http_request_duration_milliseconds",
		metric.WithDescription("Duration of HTTP requests in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return fmt.Errorf("failed to create http_request_duration histogram: %w", err)
	}

	mp.HTTPRequestsInFlight, err = mp.meter.Int64UpDownCounter(
		"http_requests_in_flight",
		metric.WithDescription("Number of HTTP requests currently being processed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create http_requests_in_flight gauge: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the metrics provider
func (mp *MetricsProvider) Shutdown(ctx context.Context) error {
	if mp.meterProvider != nil {
		return mp.meterProvider.Shutdown(ctx)
	}
	return nil
}

// FiberMiddleware returns a Fiber middleware for collecting HTTP metrics
func (mp *MetricsProvider) FiberMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		ctx := context.Background()

		// Increment in-flight requests
		mp.HTTPRequestsInFlight.Add(ctx, 1)
		defer mp.HTTPRequestsInFlight.Add(ctx, -1)

		// Process request
		err := c.Next()

		// Record metrics
		duration := time.Since(start)
		method := c.Method()
		path := c.Route().Path
		statusCode := strconv.Itoa(c.Response().StatusCode())

		// Normalize path to avoid high cardinality
		if path == "" {
			path = c.Path()
		}

		attribs := metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
			attribute.String("status_code", statusCode),
		)

		mp.HTTPRequestsTotal.Add(ctx, 1, attribs)
		mp.HTTPRequestDuration.Record(ctx, float64(duration.Milliseconds()), attribs)

		return err
	}
}

// MetricsHandler returns a Fiber handler for the /metrics endpoint
func (mp *MetricsProvider) MetricsHandler() fiber.Handler {
	// Use Fiber adaptor to convert the Prometheus HTTP handler to Fiber handler
	return adaptor.HTTPHandler(promhttp.Handler())
}

// Helper methods for TEI service metrics

// RecordEmbeddingRequest records an embedding request with duration and labels
func (mp *MetricsProvider) RecordEmbeddingRequest(ctx context.Context, duration time.Duration, protocol, status string) {
	// Record total count
	mp.EmbeddingRequestsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("protocol", protocol),
		attribute.String("status", status),
	))

	// Record duration
	mp.EmbeddingRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("protocol", protocol),
		attribute.String("status", status),
	))
}

// RecordEmbeddingError records an embedding error
func (mp *MetricsProvider) RecordEmbeddingError(ctx context.Context, protocol, errorType string) {
	mp.EmbeddingErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String("protocol", protocol),
		attribute.String("error_type", errorType),
	))
}

// UpdateGrpcConnectionStatus updates the gRPC connection status
func (mp *MetricsProvider) UpdateGrpcConnectionStatus(ctx context.Context, connected bool) {
	status := int64(0)
	if connected {
		status = 1
	}
	mp.GrpcConnectionStatus.Record(ctx, status)
}

// RecordHttpFallback records when HTTP fallback is used
func (mp *MetricsProvider) RecordHttpFallback(ctx context.Context, reason string) {
	mp.HttpFallbackCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("reason", reason),
	))
}

// RecordConnectionReconnect records a connection reconnection attempt
func (mp *MetricsProvider) RecordConnectionReconnect(ctx context.Context, success bool) {
	status := "failure"
	if success {
		status = "success"
	}
	mp.ConnectionReconnects.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}

// UpdateActiveConnections updates the active connections gauge
func (mp *MetricsProvider) UpdateActiveConnections(ctx context.Context, count int64) {
	mp.ActiveConnections.Record(ctx, count)
}

// UpdateRequestQueueSize updates the request queue size gauge
func (mp *MetricsProvider) UpdateRequestQueueSize(ctx context.Context, size int64) {
	mp.RequestQueueSize.Record(ctx, size)
}
