package main

import (
	"context"
	"fmt"
	pb "kitauji/greeter"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	serverAddr = ":55555"
)

type myServer struct {
	pb.UnimplementedGreeterServer
}

// Unary RPC
func (s *myServer) SayHelloUN(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	log.Printf("SayHelloUN: Received: %v", req)

	// テスト用に、リクエスト中の文字列に応じてエラーを返すようにしている
	switch req.GetName() {
	case "":
		return nil, fmt.Errorf("Name is blank")
	case "error":
		return nil, GetGrpcError(codes.InvalidArgument, "GRPC error")
	case "error-details":
		return nil, GetGrpcBadRequestError(codes.InvalidArgument, "GRPC error with details", "bad request")
	case "error-custom":
		return nil, GetGrpcCustomError(codes.InvalidArgument, "GRPC error with custom error", -1, "custom error")
	}

	resp := &pb.HelloResponse{
		Result: fmt.Sprintf("Hello, %s", req.Name),
	}

	return resp, nil
}

// Server streaming RPC
func (s *myServer) SayHelloSS(req *pb.HelloRequest, stream pb.Greeter_SayHelloSSServer) error {
	log.Printf("SayHelloSS: Start with a request: %v", req)

	// テスト用に、3回だけ送信したらループを抜けるようにしている
	for i := 0; i < 3; i++ {
		resp := &pb.HelloResponse{
			Result: fmt.Sprintf("Hello, %s (%d)", req.Name, i),
		}

		if err := stream.Send(resp); err != nil {
			log.Printf("SayHelloSS: Send error : %v", err)
			return err
		}

		log.Printf("SayHelloSS: Sent a message : %v", req)
		time.Sleep(1 * time.Second)
	}

	log.Printf("SayHelloSS: End")

	// ここで return すると GRPC が切断される
	return status.Errorf(codes.InvalidArgument, "Disconnect Server Streaming")
}

// Client streaming RPC
func (s *myServer) SayHelloCS(stream pb.Greeter_SayHelloCSServer) error {
	log.Printf("SayHelloCS: Start")

	for {
		// クライアントからのデータ受信待ち
		req, err := stream.Recv()
		if err != nil {
			log.Printf("SayHelloCS: Recv() error : %v", err)
			return err
		}

		log.Printf("SayHelloCS: Received: %v", req)
	}
}

// Bidirectional streaming RPC
func (s *myServer) SayHelloBI(stream pb.Greeter_SayHelloBIServer) error {
	for {
		// クライアントからのデータ受信待ち
		req, err := stream.Recv()
		if err != nil {
			log.Printf("SayHelloBI: Recv() error : %v", err)
			return err
		}

		log.Printf("SayHelloBI: Received: %v", req)

		// 受信したデータに問題がある場合
		if req.GetName() == "" {
			st := status.New(codes.InvalidArgument, "Name is blank")
			respStatus := &pb.StreamingHelloResponse_Status{Status: st.Proto()}
			resp := &pb.StreamingHelloResponse{Response: respStatus}

			stream.Send(resp)
		}

		// テスト用
		// 特定のデータが送られてきた場合は、サーバー側からも送信する
		if req.GetName() == "john" {
			respResult := &pb.StreamingHelloResponse_Result{Result: "I like john."}
			resp := &pb.StreamingHelloResponse{Response: respResult}
			stream.Send(resp)
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &myServer{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
