package tracing

import (
	"context"
	"net/http"
)

type Tracer interface {
	Init() (shutdown func(), err error)
	HTTPMiddleware(resourceID func(r *http.Request) string) func(http.Handler) http.Handler

	StartWebTransaction(ctx context.Context, name string, r *http.Request) (Transaction, context.Context)
	StartBackgroundTransaction(ctx context.Context, name string) (Transaction, context.Context)
	AddAttribute(ctx context.Context, key string, val interface{}) // requires use of StartWebTransaction or HTTPMiddleware

	BasicSegment(ctx context.Context, name string) (Segment, context.Context)
	ExternalSegment(ctx context.Context, name string, r *http.Request) (ExtSegment, context.Context)

	DatastoreSegment(ctx context.Context, subService string, hosts []string, dbName, collectionName, op string) (Segment, context.Context)
	BinstoreSegment(ctx context.Context, svc string, res string) (Segment, context.Context)
	RestSegment(ctx context.Context, operation string) (Segment, context.Context)
	JSONRPCSegment(ctx context.Context, method string) (Segment, context.Context)
	HTTPSegment(ctx context.Context, operation string, subService string, r *http.Request) (Segment, context.Context)
	DNSSegment(ctx context.Context, operation string) (Segment, context.Context)
}

type MultiTracer []Tracer

func (t MultiTracer) Init() (finisher func(), err error) {
	var fs []func()
	for _, v := range t {
		var f func()
		f, err = v.Init()
		if err != nil {
			break
		}

		fs = append(fs, f)
	}

	finisher = func() {
		for _, f := range fs {
			f()
		}
	}

	if err != nil {
		finisher()
		finisher = nil
	}

	return
}

func (t MultiTracer) HTTPMiddleware(resourceID func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		h := next
		for _, v := range t {
			h = v.HTTPMiddleware(resourceID)(h)
		}

		return h
	}
}

func (t MultiTracer) StartWebTransaction(ctx context.Context, name string, r *http.Request) (Transaction, context.Context) {
	var txn multiTransaction
	for _, v := range t {
		var t Transaction
		t, ctx = v.StartWebTransaction(ctx, name, r)
		txn = append(txn, t)
	}
	return txn, ctx
}

func (t MultiTracer) StartBackgroundTransaction(ctx context.Context, name string) (Transaction, context.Context) {
	var txn multiTransaction
	for _, v := range t {
		var t Transaction
		t, ctx = v.StartBackgroundTransaction(ctx, name)
		txn = append(txn, t)
	}
	return txn, ctx
}

func (t MultiTracer) AddAttribute(ctx context.Context, key string, val interface{}) {
	for _, v := range t {
		v.AddAttribute(ctx, key, val)
	}
}

func (t MultiTracer) BasicSegment(ctx context.Context, name string) (Segment, context.Context) {
	var seg multiSegment
	for _, v := range t {
		var s Segment
		s, ctx = v.BasicSegment(ctx, name)
		seg = append(seg, s)
	}

	return seg, ctx
}

func (t MultiTracer) ExternalSegment(ctx context.Context, name string, r *http.Request) (ExtSegment, context.Context) {
	var seg multiExtSegment
	for _, v := range t {
		var s ExtSegment
		s, ctx = v.ExternalSegment(ctx, name, r)
		seg = append(seg, s)
	}

	return seg, ctx

}

func (t MultiTracer) DatastoreSegment(ctx context.Context, subService string, hosts []string, dbName, collectionName, op string) (Segment, context.Context) {
	var seg multiSegment
	for _, v := range t {
		var s Segment
		s, ctx = v.DatastoreSegment(ctx, subService, hosts, dbName, collectionName, op)
		seg = append(seg, s)
	}

	return seg, ctx
}

func (t MultiTracer) BinstoreSegment(ctx context.Context, svc string, res string) (Segment, context.Context) {
	var seg multiSegment
	for _, v := range t {
		var s Segment
		s, ctx = v.BinstoreSegment(ctx, svc, res)
		seg = append(seg, s)
	}

	return seg, ctx
}

func (t MultiTracer) RestSegment(ctx context.Context, operation string) (Segment, context.Context) {
	var seg multiSegment
	for _, v := range t {
		var s Segment
		s, ctx = v.RestSegment(ctx, operation)
		seg = append(seg, s)
	}

	return seg, ctx
}

func (t MultiTracer) JSONRPCSegment(ctx context.Context, method string) (Segment, context.Context) {
	var seg multiSegment
	for _, v := range t {
		var s Segment
		s, ctx = v.JSONRPCSegment(ctx, method)
		seg = append(seg, s)
	}

	return seg, ctx
}

func (t MultiTracer) HTTPSegment(ctx context.Context, operation string, subService string, r *http.Request) (Segment, context.Context) {
	var seg multiSegment
	for _, v := range t {
		var s Segment
		s, ctx = v.HTTPSegment(ctx, operation, subService, r)
		seg = append(seg, s)
	}

	return seg, ctx
}

func (t MultiTracer) DNSSegment(ctx context.Context, operation string) (Segment, context.Context) {
	var seg multiSegment
	for _, v := range t {
		var s Segment
		s, ctx = v.DNSSegment(ctx, operation)
		seg = append(seg, s)
	}

	return seg, ctx
}