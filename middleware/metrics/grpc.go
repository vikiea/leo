package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"

	"github.com/go-leo/leo/middleware/internal"
	"github.com/go-leo/leo/middleware/noop"
)

// GRPCServerMiddleware 统计请求数和延迟
func GRPCServerMiddleware(opts ...Option) grpc.UnaryServerInterceptor {
	o := new(options)
	o.apply(opts...)
	o.init()
	skipMap := make(map[string]struct{}, len(o.Skips))
	for _, skip := range o.Skips {
		skipMap[skip] = struct{}{}
	}
	// 请求延迟直方图
	serverHandledHistogram, e := otel.GetMeterProvider().
		Meter(internal.InstrumentationName).
		Int64Histogram(
			"grpc.server.handling",
			metric.WithDescription("Histogram of response latency (milliseconds) of gRPC that had been application-level handled by the server."),
			metric.WithUnit("ms"),
		)
	if e != nil {
		otel.Handle(e)
		return noop.GRPCServerMiddleware()
	}
	// 请求计数器
	serverStartedCounter, e := otel.GetMeterProvider().
		Meter(internal.InstrumentationName).
		Int64Counter("grpc.server.started",
			metric.WithDescription("Total number of RPCs started on the server."),
		)
	if e != nil {
		otel.Handle(e)
		return noop.GRPCServerMiddleware()
	}
	serverHandledCounter, e := otel.GetMeterProvider().
		Meter(internal.InstrumentationName).
		Int64Counter("grpc.server.handled",
			metric.WithDescription("Total number of RPCs completed on the server, regardless of success or failure."),
		)
	if e != nil {
		otel.Handle(e)
		return noop.GRPCServerMiddleware()
	}
	serverStreamMsgReceived, e := otel.GetMeterProvider().
		Meter(internal.InstrumentationName).
		Int64Counter("grpc.server.msg.received",
			metric.WithDescription("Total number of RPC stream messages received on the server."),
		)
	if e != nil {
		otel.Handle(e)
		return noop.GRPCServerMiddleware()
	}
	serverStreamMsgSent, e := otel.GetMeterProvider().
		Meter(internal.InstrumentationName).
		Int64Counter("grpc.server.msg.sent",
			metric.WithDescription("Total number of gRPC stream messages sent by the server."),
		)
	if e != nil {
		otel.Handle(e)
		return noop.GRPCServerMiddleware()
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := skipMap[info.FullMethod]; ok {
			return handler(ctx, req)
		}
		// 开始时间
		startTime := time.Now()

		// 包含接口信息的属性
		attrs := []attribute.KeyValue{internal.GRPCType()}
		attrs = append(attrs, internal.ParseFullMethod(info.FullMethod)...)
		// 开始计数器加1
		serverStartedCounter.Add(ctx, 1, metric.WithAttributeSet(attribute.NewSet(attrs...)))
		// 接收到Msg计数器加1
		serverStreamMsgReceived.Add(ctx, 1, metric.WithAttributeSet(attribute.NewSet(attrs...)))
		// 处理一个中间件、业务逻辑
		resp, err := handler(ctx, req)
		if err == nil {
			// 发送Msg计数器加1
			serverStreamMsgSent.Add(ctx, 1, metric.WithAttributeSet(attribute.NewSet(attrs...)))
		}
		// 请求延迟直方图记录延迟
		serverHandledHistogram.Record(ctx, time.Since(startTime).Milliseconds(), metric.WithAttributeSet(attribute.NewSet(attrs...)))
		// 请求计数器加1
		serverHandledCounter.Add(ctx, 1, metric.WithAttributeSet(attribute.NewSet(append(attrs, internal.ParseError(err))...)))
		return resp, err
	}
}
