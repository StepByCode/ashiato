package appctx

import (
	"context"

	"github.com/dokkiitech/ashiato/api/internal/domain"
)

type contextKey string

const (
	actorKey   contextKey = "actor"
	traceIDKey contextKey = "trace_id"
	ipKey      contextKey = "request_ip"
)

func WithActor(ctx context.Context, actor domain.Actor) context.Context {
	return context.WithValue(ctx, actorKey, actor)
}

func ActorFromContext(ctx context.Context) (domain.Actor, bool) {
	actor, ok := ctx.Value(actorKey).(domain.Actor)
	return actor, ok
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func TraceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDKey).(string)
	return traceID
}

func WithRequestIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ipKey, ip)
}

func RequestIPFromContext(ctx context.Context) string {
	ip, _ := ctx.Value(ipKey).(string)
	return ip
}
