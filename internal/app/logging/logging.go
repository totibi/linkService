package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"
const queryStartTimeKey ctxKey = "query_start_time"

// уникальный ID для запроса
func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// HTTP Middleware: логирует входящий HTTP и пробрасывает request_id
func HttpLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Получаем или генерируем request_id
		reqID := r.Header.Get(string(requestIDKey))
		if reqID == "" {
			reqID = generateRequestID()
		}

		// Пробрасываем в контекст
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		r = r.WithContext(ctx)

		// Входящий запрос (сразу при получении)
		slog.Info("HTTP Request Received",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		// Обёртка для захвата статуса ответа
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Выполняем запрос
		next.ServeHTTP(lrw, r)

		// Завершённый запрос (после обработки)
		slog.Info("HTTP Request Completed",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// gRPC Interceptor: логирует gRPC-вызовы и извлекает request_id из метаданных
func GrpcLoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	reqID := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(string(requestIDKey)); len(vals) > 0 {
			reqID = vals[0]
		}
	}
	if reqID == "" {
		reqID = generateRequestID()
	}

	ctx = context.WithValue(ctx, requestIDKey, reqID)
	slog.Info("gRPC Call", "request_id", reqID, "method", info.FullMethod)

	start := time.Now()
	resp, err := handler(ctx, req)

	slog.Info("gRPC Response",
		"request_id", reqID,
		"method", info.FullMethod,
		"duration_ms", time.Since(start).Milliseconds(),
		"error", err,
	)
	return resp, err
}

// pgx Tracer: логирует SQL-запросы к БД
type PgxSlogTracer struct{}

// TraceQueryStart — вызывается ДО выполнения запроса
func (t *PgxSlogTracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	// Сохраняем время начала в контекст
	ctx = context.WithValue(ctx, queryStartTimeKey, time.Now())

	reqID := ctx.Value(requestIDKey)
	if reqID == nil {
		reqID = "unknown"
	}

	slog.Debug("DB Query Start",
		"request_id", reqID,
		"sql", data.SQL,
		"args", data.Args,
	)
	return ctx
}

// TraceQueryEnd — вызывается ПОСЛЕ выполнения запроса
func (t *PgxSlogTracer) TraceQueryEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	reqID := ctx.Value(requestIDKey)
	if reqID == nil {
		reqID = "unknown"
	}

	// 👇 Вычисляем длительность вручную
	var durationMs int64
	if start, ok := ctx.Value(queryStartTimeKey).(time.Time); ok {
		durationMs = time.Since(start).Milliseconds()
	}

	slog.Debug("DB Query End",
		"request_id", reqID,
		"duration_ms", durationMs,
		"err", data.Err,
	)
}
