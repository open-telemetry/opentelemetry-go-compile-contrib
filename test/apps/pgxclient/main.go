// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package main provides a minimal pgx client for integration testing.
// This client is designed to be instrumented with the otelc compile-time tool.
package main

import (
	"context"
	"flag"
	"log"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	connStr = flag.String("conn", "postgres://otelc:otelc@localhost:5432/testdb?sslmode=disable", "The PostgreSQL connection string")
	mode    = flag.String("mode", "all", "The test mode: connect, pool, or all")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	switch *mode {
	case "connect":
		runConnect(ctx)
	case "pool":
		runPool(ctx)
	case "all":
		runConnect(ctx)
		runPool(ctx)
	default:
		log.Fatalf("unknown mode: %s", *mode)
	}

	slog.Info("pgx operations completed successfully")
}

func runConnect(ctx context.Context) {
	conn, err := pgx.Connect(ctx, *connStr)
	if err != nil {
		log.Fatalf("pgx.Connect failed: %v", err)
	}
	defer conn.Close(ctx)

	var one int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		log.Fatalf("query failed: %v", err)
	}
	slog.Info("pgx.Connect successful", "result", one)
}

func runPool(ctx context.Context) {
	poolConfig, err := pgxpool.ParseConfig(*connStr)
	if err != nil {
		log.Fatalf("pgxpool.ParseConfig failed: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("pgxpool.NewWithConfig failed: %v", err)
	}
	defer pool.Close()

	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		log.Fatalf("pool query failed: %v", err)
	}
	slog.Info("pgxpool.NewWithConfig successful", "result", one)
}
