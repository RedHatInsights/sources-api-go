package main

import (
	"context"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

const tracerName = "Sources-api"

func TracerSetup() (*sdktrace.TracerProvider, error) {

	ctx := context.Background()

	options := []otlptracegrpc.Option{
		otlptracegrpc.WithInsecure()} // TODO, not for prod
	client := otlptracegrpc.NewClient(options...)
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		l.Log.Fatal(err)
		return nil, err
	}

	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exp)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
		sdktrace.WithResource(newResource()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}

func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(tracerName),
			semconv.ServiceVersionKey.String("v0.1.0"), // TODO
			attribute.String("environment", "demo"),    // TODO
		),
	)
	return r
}
