package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type databasePinger interface{ PingContext(context.Context) error }

func newRouter(db databasePinger) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	var requests atomic.Uint64
	router.Use(func(c *gin.Context) {
		started := time.Now()
		c.Next()
		requests.Add(1)
		if c.Request.URL.Path != "/metrics" {
			slog.Info("request", "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration", time.Since(started))
		}
	})
	router.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"service": Settings.AppName, "status": "ok"}) })
	router.GET("/api/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy"}) })
	router.GET("/api/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		if db == nil || db.PingContext(ctx) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "database": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready", "database": "connected"})
	})
	router.GET("/metrics", func(c *gin.Context) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, "# TYPE service_http_requests_total counter\nservice_http_requests_total %d\n# TYPE service_go_heap_bytes gauge\nservice_go_heap_bytes %d\n# TYPE service_go_goroutines gauge\nservice_go_goroutines %d\n", requests.Load(), mem.HeapAlloc, runtime.NumGoroutine())
	})
	return router
}

func main() {
	_ = godotenv.Load()
	InitSettings()
	initLogging()
	var db *sql.DB
	if Settings.DatabaseURL != "" {
		var err error
		db, err = sql.Open("postgres", Settings.DatabaseURL)
		if err != nil {
			slog.Error("invalid database configuration")
			os.Exit(1)
		}
		db.SetMaxOpenConns(2)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(5 * time.Minute)
		defer db.Close()
	}
	var pinger databasePinger
	if db != nil {
		pinger = db
	}
	server := &http.Server{Addr: fmt.Sprintf("%s:%d", Settings.Host, Settings.Port), Handler: newRouter(pinger), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		slog.Info("listening", "address", server.Addr, "service", Settings.AppName)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("HTTP shutdown failed", "error", err)
	}
}
