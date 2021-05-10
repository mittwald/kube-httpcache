package tracing

import (
	"context"
	"fmt"
	"github.com/newrelic/go-agent/v3/integrations/nrmongo"
	"go.mongodb.org/mongo-driver/event"
	mongotrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go.mongodb.org/mongo-driver/mongo"
	"net/http"
)

var tracers MultiTracer
var _ Tracer = tracers // make sure it satisfies the interface

func RegisterTracer(t Tracer) {
	tracers = append(tracers, t)
}

func Init() (func(), error) {
	return tracers.Init()
}

func MongoDriverMonitor(service string) *event.CommandMonitor {
	monitor := mongotrace.NewMonitor(
		mongotrace.WithServiceName(service + "-mongodb"),
	)

	monitor = nrmongo.NewCommandMonitor(monitor)

	return monitor
}

func HTTPMiddleware(resourceID func(r *http.Request) string) func(http.Handler) http.Handler {
	return tracers.HTTPMiddleware(resourceID)
}

func StartWebTransaction(ctx context.Context, name string, r *http.Request) (Transaction, context.Context) {
	return tracers.StartWebTransaction(ctx, name, r)
}

func StartBackgroundTransaction(ctx context.Context, name string) (Transaction, context.Context) {
	return tracers.StartBackgroundTransaction(ctx, name)
}

func AddAttribute(ctx context.Context, key string, val interface{}) {
	tracers.AddAttribute(ctx, key, val)
}

func BasicSegment(ctx context.Context, name string) (Segment, context.Context) {
	return tracers.BasicSegment(ctx, name)
}

func ExternalSegment(ctx context.Context, name string, r *http.Request) (ExtSegment, context.Context) {
	return tracers.ExternalSegment(ctx, name, r)
}

func DatastoreSegment(ctx context.Context, subService string, hosts []string, dbName, collectionName, op string) (Segment, context.Context) {
	return tracers.DatastoreSegment(ctx, subService, hosts, dbName, collectionName, op)
}

func BinstoreSegment(ctx context.Context, adapter string, op string, loc string) (Segment, context.Context) {
	subSvc := fmt.Sprintf("binarystore-%s", adapter)

	res := op
	if loc != "" {
		res = fmt.Sprintf("%s %s", op, loc)
	}
	return tracers.BinstoreSegment(ctx, subSvc, res)
}

func RestSegment(ctx context.Context, operation string) (Segment, context.Context) {
	return tracers.RestSegment(ctx, operation)
}

func JSONRPCSegment(ctx context.Context, method string) (Segment, context.Context) {
	return tracers.JSONRPCSegment(ctx, method)
}

func HTTPSegment(ctx context.Context, operation string, subService string, r *http.Request) (Segment, context.Context) {
	return tracers.HTTPSegment(ctx, operation, subService, r)
}

func DNSSegment(ctx context.Context, operation string) (Segment, context.Context) {
	return tracers.DNSSegment(ctx, operation)
}
