package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"traffic-generator/internal/telemetry"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type SimulationStats struct {
	StartTime time.Time

	TotalRequests  int
	FailedRequests int

	UniqueUsers    map[string]struct{}
	UniqueProducts map[string]struct{}
}

func main() {

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Println("Connecting to SigNoz OpenTelemetry Collector...")

	tp, err := telemetry.InitTracer(
		ctx,
		"checkout-gateway-service",
	)

	if err != nil {
		log.Fatalf(
			"Failed to initialize OpenTelemetry tracer: %v",
			err,
		)
	}

	stats := &SimulationStats{
		StartTime:      time.Now(),
		UniqueUsers:    make(map[string]struct{}),
		UniqueProducts: make(map[string]struct{}),
	}

	defer func() {

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := tp.Shutdown(shutdownCtx); err != nil {
			log.Printf(
				"Error shutting down tracer provider: %v",
				err,
			)
		} else {
			log.Println(
				"Tracer Provider safely flushed and shut down.",
			)
		}

	}()

	defer printSummary(stats)

	log.Println(
		"OpenTelemetry pipeline successfully wired to SigNoz backend!",
	)

	tracer := otel.Tracer(
		"warmup-traffic-generator",
	)

	log.Println(
		"Starting synthetic traffic simulation. Press Ctrl+C to terminate early.",
	)

Loop:

	for i := 1; i <= 5000; i++ {

		select {

		case <-ctx.Done():

			log.Println(
				"Termination signal captured. Stopping execution loop...",
			)

			break Loop

		default:

			generateMockTransaction(
				ctx,
				tracer,
				i,
				stats,
			)

			time.Sleep(
				50 * time.Millisecond,
			)
		}
	}

	log.Println(
		"Traffic execution loop completed.",
	)
}

func generateMockTransaction(
	ctx context.Context,
	tracer trace.Tracer,
	requestID int,
	stats *SimulationStats,
) {

	ctx, rootSpan := tracer.Start(
		ctx,
		"HTTP POST /checkout",
		trace.WithSpanKind(trace.SpanKindServer),
	)

	defer rootSpan.End()

	fakeUserID := fmt.Sprintf(
		"usr-%06d-%d",
		requestID,
		rand.IntN(1000),
	)

	fakeProductID := fmt.Sprintf(
		"prod-%016x",
		rand.Uint64(),
	)

	stats.TotalRequests++

	stats.UniqueUsers[fakeUserID] = struct{}{}
	stats.UniqueProducts[fakeProductID] = struct{}{}

	rootSpan.SetAttributes(
		attribute.String(
			"http.method",
			"POST",
		),

		attribute.Int(
			"request.id",
			requestID,
		),

		attribute.String(
			"user.id",
			fakeUserID,
		),

		attribute.String(
			"product.id",
			fakeProductID,
		),
	)

	_, dbSpan := tracer.Start(
		ctx,
		"SQL SELECT clusters_meta",
		trace.WithSpanKind(trace.SpanKindClient),
	)

	defer dbSpan.End()

	time.Sleep(
		time.Duration(
			rand.IntN(30)+5,
		) * time.Millisecond,
	)

	if requestID%15 == 0 {

		stats.FailedRequests++

		err := fmt.Errorf(
			"database connection timeout on pool allocation",
		)

		dbSpan.RecordError(err)

		dbSpan.SetStatus(
			codes.Error,
			err.Error(),
		)

		rootSpan.SetAttributes(
			attribute.Int(
				"http.status_code",
				500,
			),
		)

		rootSpan.SetStatus(
			codes.Error,
			"checkout failed",
		)

		rootSpan.AddEvent(
			"checkout_failure",
			trace.WithAttributes(

				attribute.String(
					"error.type",
					"database_timeout",
				),

				attribute.String(
					"error.message",
					err.Error(),
				),

				attribute.String(
					"user.id",
					fakeUserID,
				),

				attribute.String(
					"product.id",
					fakeProductID,
				),
			),
		)

		log.Printf(
			"[ERROR] Request ID %d failed for User %s processing product %s",
			requestID,
			fakeUserID,
			fakeProductID,
		)

	} else {

		rootSpan.SetAttributes(
			attribute.Int(
				"http.status_code",
				200,
			),
		)

		rootSpan.SetStatus(
			codes.Ok,
			"checkout completed",
		)

	}

}

func printSummary(
	stats *SimulationStats,
) {

	duration := time.Since(
		stats.StartTime,
	)

	rps := float64(
		stats.TotalRequests,
	) / duration.Seconds()

	fmt.Println()

	fmt.Println(
		"========== Simulation Summary ==========",
	)

	fmt.Printf(
		"Duration           : %v\n",
		duration.Round(time.Second),
	)

	fmt.Printf(
		"Requests Sent      : %d\n",
		stats.TotalRequests,
	)

	fmt.Printf(
		"Injected Failures  : %d\n",
		stats.FailedRequests,
	)

	fmt.Printf(
		"Unique Users       : %d\n",
		len(stats.UniqueUsers),
	)

	fmt.Printf(
		"Unique Products    : %d\n",
		len(stats.UniqueProducts),
	)

	fmt.Printf(
		"Average Throughput : %.2f requests/sec\n",
		rps,
	)

	fmt.Println(
		"========================================",
	)

}
