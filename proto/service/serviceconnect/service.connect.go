// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: service.proto

package serviceconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	service "github.com/fr13n8/raido/proto/service"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// RaidoServiceName is the fully-qualified name of the RaidoService service.
	RaidoServiceName = "service.RaidoService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// RaidoServiceProxyStartProcedure is the fully-qualified name of the RaidoService's ProxyStart RPC.
	RaidoServiceProxyStartProcedure = "/service.RaidoService/ProxyStart"
	// RaidoServiceProxyStopProcedure is the fully-qualified name of the RaidoService's ProxyStop RPC.
	RaidoServiceProxyStopProcedure = "/service.RaidoService/ProxyStop"
	// RaidoServiceGetAgentsProcedure is the fully-qualified name of the RaidoService's GetAgents RPC.
	RaidoServiceGetAgentsProcedure = "/service.RaidoService/GetAgents"
	// RaidoServiceAgentTunnelStartProcedure is the fully-qualified name of the RaidoService's
	// AgentTunnelStart RPC.
	RaidoServiceAgentTunnelStartProcedure = "/service.RaidoService/AgentTunnelStart"
	// RaidoServiceAgentTunnelStopProcedure is the fully-qualified name of the RaidoService's
	// AgentTunnelStop RPC.
	RaidoServiceAgentTunnelStopProcedure = "/service.RaidoService/AgentTunnelStop"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	raidoServiceServiceDescriptor                = service.File_service_proto.Services().ByName("RaidoService")
	raidoServiceProxyStartMethodDescriptor       = raidoServiceServiceDescriptor.Methods().ByName("ProxyStart")
	raidoServiceProxyStopMethodDescriptor        = raidoServiceServiceDescriptor.Methods().ByName("ProxyStop")
	raidoServiceGetAgentsMethodDescriptor        = raidoServiceServiceDescriptor.Methods().ByName("GetAgents")
	raidoServiceAgentTunnelStartMethodDescriptor = raidoServiceServiceDescriptor.Methods().ByName("AgentTunnelStart")
	raidoServiceAgentTunnelStopMethodDescriptor  = raidoServiceServiceDescriptor.Methods().ByName("AgentTunnelStop")
)

// RaidoServiceClient is a client for the service.RaidoService service.
type RaidoServiceClient interface {
	ProxyStart(context.Context, *connect.Request[service.ProxyStartRequest]) (*connect.Response[service.ProxyStartResponse], error)
	ProxyStop(context.Context, *connect.Request[service.ProxyStopRequest]) (*connect.Response[service.ProxyStopResponse], error)
	GetAgents(context.Context, *connect.Request[service.GetAgentsRequest]) (*connect.Response[service.GetAgentsResponse], error)
	AgentTunnelStart(context.Context, *connect.Request[service.AgentTunnelStartRequest]) (*connect.Response[service.AgentTunnelStartResponse], error)
	AgentTunnelStop(context.Context, *connect.Request[service.AgentTunnelStopRequest]) (*connect.Response[service.AgentTunnelStopResponse], error)
}

// NewRaidoServiceClient constructs a client for the service.RaidoService service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewRaidoServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) RaidoServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &raidoServiceClient{
		proxyStart: connect.NewClient[service.ProxyStartRequest, service.ProxyStartResponse](
			httpClient,
			baseURL+RaidoServiceProxyStartProcedure,
			connect.WithSchema(raidoServiceProxyStartMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		proxyStop: connect.NewClient[service.ProxyStopRequest, service.ProxyStopResponse](
			httpClient,
			baseURL+RaidoServiceProxyStopProcedure,
			connect.WithSchema(raidoServiceProxyStopMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		getAgents: connect.NewClient[service.GetAgentsRequest, service.GetAgentsResponse](
			httpClient,
			baseURL+RaidoServiceGetAgentsProcedure,
			connect.WithSchema(raidoServiceGetAgentsMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		agentTunnelStart: connect.NewClient[service.AgentTunnelStartRequest, service.AgentTunnelStartResponse](
			httpClient,
			baseURL+RaidoServiceAgentTunnelStartProcedure,
			connect.WithSchema(raidoServiceAgentTunnelStartMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		agentTunnelStop: connect.NewClient[service.AgentTunnelStopRequest, service.AgentTunnelStopResponse](
			httpClient,
			baseURL+RaidoServiceAgentTunnelStopProcedure,
			connect.WithSchema(raidoServiceAgentTunnelStopMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// raidoServiceClient implements RaidoServiceClient.
type raidoServiceClient struct {
	proxyStart       *connect.Client[service.ProxyStartRequest, service.ProxyStartResponse]
	proxyStop        *connect.Client[service.ProxyStopRequest, service.ProxyStopResponse]
	getAgents        *connect.Client[service.GetAgentsRequest, service.GetAgentsResponse]
	agentTunnelStart *connect.Client[service.AgentTunnelStartRequest, service.AgentTunnelStartResponse]
	agentTunnelStop  *connect.Client[service.AgentTunnelStopRequest, service.AgentTunnelStopResponse]
}

// ProxyStart calls service.RaidoService.ProxyStart.
func (c *raidoServiceClient) ProxyStart(ctx context.Context, req *connect.Request[service.ProxyStartRequest]) (*connect.Response[service.ProxyStartResponse], error) {
	return c.proxyStart.CallUnary(ctx, req)
}

// ProxyStop calls service.RaidoService.ProxyStop.
func (c *raidoServiceClient) ProxyStop(ctx context.Context, req *connect.Request[service.ProxyStopRequest]) (*connect.Response[service.ProxyStopResponse], error) {
	return c.proxyStop.CallUnary(ctx, req)
}

// GetAgents calls service.RaidoService.GetAgents.
func (c *raidoServiceClient) GetAgents(ctx context.Context, req *connect.Request[service.GetAgentsRequest]) (*connect.Response[service.GetAgentsResponse], error) {
	return c.getAgents.CallUnary(ctx, req)
}

// AgentTunnelStart calls service.RaidoService.AgentTunnelStart.
func (c *raidoServiceClient) AgentTunnelStart(ctx context.Context, req *connect.Request[service.AgentTunnelStartRequest]) (*connect.Response[service.AgentTunnelStartResponse], error) {
	return c.agentTunnelStart.CallUnary(ctx, req)
}

// AgentTunnelStop calls service.RaidoService.AgentTunnelStop.
func (c *raidoServiceClient) AgentTunnelStop(ctx context.Context, req *connect.Request[service.AgentTunnelStopRequest]) (*connect.Response[service.AgentTunnelStopResponse], error) {
	return c.agentTunnelStop.CallUnary(ctx, req)
}

// RaidoServiceHandler is an implementation of the service.RaidoService service.
type RaidoServiceHandler interface {
	ProxyStart(context.Context, *connect.Request[service.ProxyStartRequest]) (*connect.Response[service.ProxyStartResponse], error)
	ProxyStop(context.Context, *connect.Request[service.ProxyStopRequest]) (*connect.Response[service.ProxyStopResponse], error)
	GetAgents(context.Context, *connect.Request[service.GetAgentsRequest]) (*connect.Response[service.GetAgentsResponse], error)
	AgentTunnelStart(context.Context, *connect.Request[service.AgentTunnelStartRequest]) (*connect.Response[service.AgentTunnelStartResponse], error)
	AgentTunnelStop(context.Context, *connect.Request[service.AgentTunnelStopRequest]) (*connect.Response[service.AgentTunnelStopResponse], error)
}

// NewRaidoServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewRaidoServiceHandler(svc RaidoServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	raidoServiceProxyStartHandler := connect.NewUnaryHandler(
		RaidoServiceProxyStartProcedure,
		svc.ProxyStart,
		connect.WithSchema(raidoServiceProxyStartMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	raidoServiceProxyStopHandler := connect.NewUnaryHandler(
		RaidoServiceProxyStopProcedure,
		svc.ProxyStop,
		connect.WithSchema(raidoServiceProxyStopMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	raidoServiceGetAgentsHandler := connect.NewUnaryHandler(
		RaidoServiceGetAgentsProcedure,
		svc.GetAgents,
		connect.WithSchema(raidoServiceGetAgentsMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	raidoServiceAgentTunnelStartHandler := connect.NewUnaryHandler(
		RaidoServiceAgentTunnelStartProcedure,
		svc.AgentTunnelStart,
		connect.WithSchema(raidoServiceAgentTunnelStartMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	raidoServiceAgentTunnelStopHandler := connect.NewUnaryHandler(
		RaidoServiceAgentTunnelStopProcedure,
		svc.AgentTunnelStop,
		connect.WithSchema(raidoServiceAgentTunnelStopMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/service.RaidoService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case RaidoServiceProxyStartProcedure:
			raidoServiceProxyStartHandler.ServeHTTP(w, r)
		case RaidoServiceProxyStopProcedure:
			raidoServiceProxyStopHandler.ServeHTTP(w, r)
		case RaidoServiceGetAgentsProcedure:
			raidoServiceGetAgentsHandler.ServeHTTP(w, r)
		case RaidoServiceAgentTunnelStartProcedure:
			raidoServiceAgentTunnelStartHandler.ServeHTTP(w, r)
		case RaidoServiceAgentTunnelStopProcedure:
			raidoServiceAgentTunnelStopHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedRaidoServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedRaidoServiceHandler struct{}

func (UnimplementedRaidoServiceHandler) ProxyStart(context.Context, *connect.Request[service.ProxyStartRequest]) (*connect.Response[service.ProxyStartResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("service.RaidoService.ProxyStart is not implemented"))
}

func (UnimplementedRaidoServiceHandler) ProxyStop(context.Context, *connect.Request[service.ProxyStopRequest]) (*connect.Response[service.ProxyStopResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("service.RaidoService.ProxyStop is not implemented"))
}

func (UnimplementedRaidoServiceHandler) GetAgents(context.Context, *connect.Request[service.GetAgentsRequest]) (*connect.Response[service.GetAgentsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("service.RaidoService.GetAgents is not implemented"))
}

func (UnimplementedRaidoServiceHandler) AgentTunnelStart(context.Context, *connect.Request[service.AgentTunnelStartRequest]) (*connect.Response[service.AgentTunnelStartResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("service.RaidoService.AgentTunnelStart is not implemented"))
}

func (UnimplementedRaidoServiceHandler) AgentTunnelStop(context.Context, *connect.Request[service.AgentTunnelStopRequest]) (*connect.Response[service.AgentTunnelStopResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("service.RaidoService.AgentTunnelStop is not implemented"))
}
