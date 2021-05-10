package tracing

import (
	"context"
	"fmt"
	"github.com/newrelic/go-agent/v3/newrelic"
	"net/http"
	"os"
	"strings"
	"time"
)

type nrTracer struct {
	app *newrelic.Application
}

func nrSegment(txn *newrelic.Transaction, addAttribute func(key string, val interface{}), setResponse func(r *http.Response), end func()) ExtSegment {
	return segment{
		setAttribute: addAttribute,
		setResponse:  setResponse,
		finish: func(err error) {
			if err != nil {
				txn.NoticeError(err)
			}
			end()
		},
	}
}

func (t *nrTracer) Init() (shutdown func(), err error) {
	t.app, err = newrelic.NewApplication(newrelic.ConfigFromEnvironment())
	if err != nil {
		return
	}

	shutdown = func() {
		t.app.Shutdown(10 * time.Second)
	}

	return
}

func (t nrTracer) HTTPMiddleware(resourceID func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			txn := t.app.StartTransaction("/")
			defer txn.End()

			w = txn.SetWebResponse(w)
			txn.SetWebRequestHTTP(r)

			r = newrelic.RequestWithTransactionContext(r, txn)

			next.ServeHTTP(w, r)

			txn.SetName(resourceID(r))
		})
	}
}

func (t nrTracer) StartWebTransaction(ctx context.Context, name string, r *http.Request) (Transaction, context.Context) {
	txn := t.app.StartTransaction(name)
	txn.SetWebRequestHTTP(r)

	return transaction{
		finish: func(err error) {
			if err != nil {
				txn.NoticeError(err)
			}
			txn.End()
		},
	}, newrelic.NewContext(ctx, txn)
}

func (t nrTracer) StartBackgroundTransaction(ctx context.Context, name string) (Transaction, context.Context) {
	txn := t.app.StartTransaction(name)
	return transaction{
		finish: func(err error) {
			if err != nil {
				txn.NoticeError(err)
			}
			txn.End()
		},
	}, newrelic.NewContext(ctx, txn)
}

func (t nrTracer) AddAttribute(ctx context.Context, key string, val interface{}) {
	txn := newrelic.FromContext(ctx)
	txn.AddAttribute(key, val)
}

func (_ nrTracer) BasicSegment(ctx context.Context, name string) (Segment, context.Context) {
	txn := newrelic.FromContext(ctx)
	seg := &newrelic.Segment{
		StartTime: txn.StartSegmentNow(),
		Name:      name,
	}
	return nrSegment(txn, seg.AddAttribute, func(response *http.Response) {}, seg.End), ctx
}

func (_ nrTracer) ExternalSegment(ctx context.Context, name string, r *http.Request) (ExtSegment, context.Context) {
	txn := newrelic.FromContext(ctx)
	seg := newrelic.StartExternalSegment(txn, r)

	return nrSegment(txn, seg.AddAttribute, func(response *http.Response) {
		seg.Response = response
	}, seg.End), ctx
}

func (_ nrTracer) DatastoreSegment(ctx context.Context, subService string, hosts []string, dbName, collectionName, op string) (Segment, context.Context) {
	txn := newrelic.FromContext(ctx)
	seg := &newrelic.DatastoreSegment{
		StartTime:    txn.StartSegmentNow(),
		Product:      newrelic.DatastoreMongoDB,
		Collection:   collectionName,
		Operation:    op,
		Host:         strings.Join(hosts, ","),
		DatabaseName: dbName,
	}

	return nrSegment(txn, seg.AddAttribute, func(response *http.Response) {}, seg.End), ctx
}

func (_ nrTracer) BinstoreSegment(ctx context.Context, svc string, res string) (Segment, context.Context) {
	txn := newrelic.FromContext(ctx)
	seg := &newrelic.Segment{
		StartTime: txn.StartSegmentNow(),
		Name:      fmt.Sprintf("%s:%s", svc, res),
	}

	return nrSegment(txn, seg.AddAttribute, func(response *http.Response) {}, seg.End), ctx
}

func (_ nrTracer) RestSegment(ctx context.Context, operation string) (Segment, context.Context) {
	txn := newrelic.FromContext(ctx)
	seg := &newrelic.Segment{
		StartTime: txn.StartSegmentNow(),
		Name:      operation,
	}

	return nrSegment(txn, seg.AddAttribute, func(response *http.Response) {}, seg.End), ctx
}

func (_ nrTracer) JSONRPCSegment(ctx context.Context, method string) (Segment, context.Context) {
	txn := newrelic.FromContext(ctx)
	seg := &newrelic.Segment{
		StartTime: txn.StartSegmentNow(),
		Name:      "jsonrpc." + method,
	}

	return nrSegment(txn, seg.AddAttribute, func(response *http.Response) {}, seg.End), ctx
}

func (_ nrTracer) HTTPSegment(ctx context.Context, operation string, subService string, r *http.Request) (Segment, context.Context) {
	txn := newrelic.FromContext(ctx)
	seg := newrelic.StartExternalSegment(txn, r)

	return nrSegment(txn, seg.AddAttribute, func(response *http.Response) {}, seg.End), ctx
}

func (_ nrTracer) DNSSegment(ctx context.Context, operation string) (Segment, context.Context) {
	txn := newrelic.FromContext(ctx)
	seg := &newrelic.Segment{
		StartTime: txn.StartSegmentNow(),
		Name:      operation,
	}

	return nrSegment(txn, seg.AddAttribute, func(response *http.Response) {}, seg.End), ctx
}

func init() {
	if os.Getenv("NEW_RELIC_LICENSE_KEY") != "" {
		RegisterTracer(&nrTracer{})
	}
}
