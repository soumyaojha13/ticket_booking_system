package proto

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SeatRequest struct {
	SeatId string
	UserId string
}

type SeatResponse struct {
	Status string
}

type SeatServiceServer interface {
	LockSeat(context.Context, *SeatRequest) (*SeatResponse, error)
	ReleaseSeat(context.Context, *SeatRequest) (*SeatResponse, error)
}

type UnimplementedSeatServiceServer struct{}

func (*UnimplementedSeatServiceServer) LockSeat(context.Context, *SeatRequest) (*SeatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LockSeat not implemented")
}

func (*UnimplementedSeatServiceServer) ReleaseSeat(context.Context, *SeatRequest) (*SeatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReleaseSeat not implemented")
}

func RegisterSeatServiceServer(s *grpc.Server, srv SeatServiceServer) {
	s.RegisterService(&_SeatService_serviceDesc, srv)
}

var _SeatService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "proto.SeatService",
	HandlerType: (*SeatServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "LockSeat",
			Handler:    _SeatService_LockSeat_Handler,
		},
		{
			MethodName: "ReleaseSeat",
			Handler:    _SeatService_ReleaseSeat_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "",
}

func _SeatService_LockSeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SeatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SeatServiceServer).LockSeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.SeatService/LockSeat",
	}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SeatServiceServer).LockSeat(ctx, req.(*SeatRequest))
	})
}

func _SeatService_ReleaseSeat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SeatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SeatServiceServer).ReleaseSeat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.SeatService/ReleaseSeat",
	}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SeatServiceServer).ReleaseSeat(ctx, req.(*SeatRequest))
	})
}
