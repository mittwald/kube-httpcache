package tracing

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func ddServiceName(sub string) string {
	svc := os.Getenv("DD_SERVICE")

	if sub != "" {
		return fmt.Sprintf("%s-%s", svc, sub)
	}

	return svc
}

func ddSegment(span tracer.Span) ExtSegment {
	return segment{
		setAttribute: span.SetTag,
		setResponse: func(r *http.Response) {
			span.SetTag(ext.HTTPCode, strconv.Itoa(r.StatusCode))
		},
		finish: func(err error) { span.Finish(tracer.WithError(err)) },
	}
}

type ddTracer struct{}

func (_ ddTracer) Init() (shutdown func(), err error) {
	service := ddServiceName("")
	tracer.Start(tracer.WithService(service))

	return tracer.Stop, nil
}

func (_ ddTracer) HTTPMiddleware(resourceID func(r *http.Request) string) func(http.Handler) http.Handler {
	// This bullshit is because while the global tracer will default service name from env, it provides us no
	// means of getting the name for our httptrace setup below.
	service := ddServiceName("")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Certain libraries like go-chi (and probably others) don't resolve
			// routes until after the middlewares have already executed. We want to
			// set up tracing immediately but wait to call the `resourceID` callback
			// after the request as processed. Since datadog stores the current span
			// in the request context, we make a small handler here that can pull
			// the span out of the context in order to set the resource name at the
			// end of the request.

			// httptrace.WrapHandler     -- create generic span, call h
			//   -> h
			//      -> next              -- finish processing the request (this includes routing)
			//      -> get generic span from context
			//      -> resourceID(r)     -- ask the caller for the resource id
			//      -> span.SetTag       -- assign the resource id to the span
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
				if span, ok := tracer.SpanFromContext(r.Context()); ok {
					span.SetTag(ext.ResourceName, resourceID(r))
				}
			})

			httptrace.WrapHandler(h, service, "/").ServeHTTP(w, r)
		})
	}
}

func (t ddTracer) StartWebTransaction(ctx context.Context, name string, r *http.Request) (Transaction, context.Context) {
	return t.BasicSegment(ctx, name)
}

func (t ddTracer) StartBackgroundTransaction(ctx context.Context, name string) (Transaction, context.Context) {
	return t.BasicSegment(ctx, name)
}

func (t ddTracer) AddAttribute(ctx context.Context, key string, val interface{}) {
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		span.SetTag(key, val)
	}
}

func (_ ddTracer) BasicSegment(ctx context.Context, name string) (Segment, context.Context) {
	span, ctx := tracer.StartSpanFromContext(ctx, name)
	return ddSegment(span), ctx
}

func (_ ddTracer) ExternalSegment(ctx context.Context, name string, r *http.Request) (ExtSegment, context.Context) {
	span, ctx := tracer.StartSpanFromContext(
		ctx,
		name,
		tracer.SpanType(ext.SpanTypeHTTP),
		tracer.ServiceName(ddServiceName("http")),
		tracer.Tag(ext.HTTPMethod, r.Method),
		tracer.Tag(ext.HTTPURL, "http://"+r.Host+r.URL.Path),
	)

	err := tracer.Inject(span.Context(), tracer.HTTPHeadersCarrier(r.Header))
	if err != nil {
		// This is instrumentation, so just log the error and continue
		err = fmt.Errorf("tracer.Inject error: %v", err)
		logrus.Error(err)
	}

	return ddSegment(span), ctx}

func (_ ddTracer) DatastoreSegment(ctx context.Context, subService string, hosts []string, dbName string, collectionName string, op string) (Segment, context.Context) {
	span, ctx := tracer.StartSpanFromContext(
		ctx,
		"mongodb."+op,
		tracer.SpanType(ext.SpanTypeMongoDB),
		tracer.ServiceName(ddServiceName(subService)),
		tracer.ResourceName(op),
		tracer.Tag("collection", collectionName),
		tracer.Tag("database", dbName),
		tracer.Tag("database_hosts", strings.Join(hosts, ",")),
		tracer.Tag("product", "mongodb"),
	)

	return ddSegment(span), ctx
}

func (_ ddTracer) BinstoreSegment(ctx context.Context, svc string, res string) (Segment, context.Context) {
	span, ctx := tracer.StartSpanFromContext(
		ctx,
		fmt.Sprintf("%s:%s", svc, res),
		tracer.SpanType(ext.SpanTypeHTTP),
		tracer.ServiceName(ddServiceName(svc)),
		tracer.ResourceName(res),
	)

	return ddSegment(span), ctx
}

func (_ ddTracer) RestSegment(ctx context.Context, operation string) (Segment, context.Context) {
	span, ctx := tracer.StartSpanFromContext(
		ctx,
		operation,
		tracer.SpanType(ext.SpanTypeWeb),
	)
	return ddSegment(span), ctx
}

func (_ ddTracer) JSONRPCSegment(ctx context.Context, method string) (Segment, context.Context) {
	span, ctx := tracer.StartSpanFromContext(
		ctx,
		"jsonrpc."+method,
		tracer.SpanType(ext.SpanTypeWeb),
		tracer.ServiceName(ddServiceName("jsonrpc")),
		tracer.ResourceName(method),
		tracer.Tag("method", method),
	)

	return ddSegment(span), ctx
}

func (_ ddTracer) HTTPSegment(ctx context.Context, operation string, subService string, r *http.Request) (Segment, context.Context) {
	span, ctx := tracer.StartSpanFromContext(
		ctx,
		operation,
		tracer.SpanType(ext.SpanTypeElasticSearch),
		tracer.ServiceName(ddServiceName(subService)),
		tracer.ResourceName(fmt.Sprintf("POST %s", r.URL.Path)),
	)

	err := tracer.Inject(span.Context(), tracer.HTTPHeadersCarrier(r.Header))
	if err != nil {
		// This is instrumentation, so just log the error and continue
		err = fmt.Errorf("tracer.Inject error: %v", err)
		logrus.Error(err)
	}

	return ddSegment(span), ctx
}

func (_ ddTracer) DNSSegment(ctx context.Context, operation string) (Segment, context.Context) {
	span, ctx := tracer.StartSpanFromContext(ctx, operation,
		tracer.SpanType(ext.SpanTypeDNS),
		tracer.ServiceName(ddServiceName("dns")),
	)
	return ddSegment(span), ctx
}

func init() {
	if os.Getenv("DD_AGENT_HOST") != "" && os.Getenv("DD_SERVICE") != "" {
		RegisterTracer(&ddTracer{})
	}
}
