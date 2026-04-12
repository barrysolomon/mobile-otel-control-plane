// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type LogExporter struct {
	provider  *sdklog.LoggerProvider
	logger    otellog.Logger
	authToken string
}

// MobileEvent represents an event received from the mobile device
type MobileEvent struct {
	EventName     string                 `json:"event_name"`
	SessionID     string                 `json:"session_id"`
	DeviceID      string                 `json:"device_id"`
	TriggerID     string                 `json:"trigger_id,omitempty"`
	ConfigVersion int                    `json:"config_version"`
	Timestamp     int64                  `json:"timestamp"` // Unix timestamp in milliseconds
	Attributes    map[string]interface{} `json:"attributes"`
}

func NewLogExporter(ctx context.Context, collectorEndpoint string, authToken string) (*LogExporter, error) {
	// Create gRPC connection to collector.
	// Default: insecure (matches K8s/Docker Compose where collector has no TLS on :4317).
	// Set COLLECTOR_TLS_ENABLED=true for production endpoints that require TLS (e.g., Dash0 ingress).
	var transportCreds grpc.DialOption
	if os.Getenv("COLLECTOR_TLS_ENABLED") == "true" {
		transportCreds = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
		log.Println("gRPC: TLS enabled for collector connection")
	} else {
		transportCreds = grpc.WithTransportCredentials(insecure.NewCredentials())
		log.Println("gRPC: insecure (no TLS) — set COLLECTOR_TLS_ENABLED=true for production")
	}
	conn, err := grpc.NewClient(collectorEndpoint, transportCreds)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	// Create OTLP log exporter with optional auth headers
	exporterOpts := []otlploggrpc.Option{
		otlploggrpc.WithGRPCConn(conn),
	}

	// Add authorization header if token is provided
	if authToken != "" {
		exporterOpts = append(exporterOpts,
			otlploggrpc.WithHeaders(map[string]string{
				"Authorization": "Bearer " + authToken,
			}),
		)
	}

	exporter, err := otlploggrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("mobile-observability-gateway"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create logger provider
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	// Get logger
	logger := provider.Logger("gateway")

	return &LogExporter{
		provider:  provider,
		logger:    logger,
		authToken: authToken,
	}, nil
}

func (e *LogExporter) ExportEvents(ctx context.Context, events []MobileEvent) error {
	for _, event := range events {
		if err := e.exportEvent(ctx, event); err != nil {
			return fmt.Errorf("failed to export event: %w", err)
		}
	}
	return nil
}

func (e *LogExporter) exportEvent(ctx context.Context, event MobileEvent) error {
	// Convert timestamp from milliseconds to time.Time
	timestamp := time.UnixMilli(event.Timestamp)

	// Build log record
	var record otellog.Record
	record.SetTimestamp(timestamp)
	record.SetBody(otellog.StringValue(event.EventName))

	// Add standard attributes
	record.AddAttributes(
		otellog.String("session_id", event.SessionID),
		otellog.String("device_id", event.DeviceID),
		otellog.Int("config_version", event.ConfigVersion),
	)

	// Add trigger_id if present
	if event.TriggerID != "" {
		record.AddAttributes(otellog.String("trigger_id", event.TriggerID))
	}

	// Add custom attributes from event
	for key, value := range event.Attributes {
		record.AddAttributes(convertAttribute(key, value))
	}

	// Emit log
	e.logger.Emit(ctx, record)

	return nil
}

func convertAttribute(key string, value interface{}) otellog.KeyValue {
	switch v := value.(type) {
	case string:
		return otellog.String(key, v)
	case int:
		return otellog.Int(key, v)
	case int64:
		return otellog.Int64(key, v)
	case float64:
		return otellog.Float64(key, v)
	case bool:
		return otellog.Bool(key, v)
	default:
		// Fallback to string representation
		return otellog.String(key, fmt.Sprintf("%v", v))
	}
}

func (e *LogExporter) Shutdown(ctx context.Context) error {
	return e.provider.Shutdown(ctx)
}
