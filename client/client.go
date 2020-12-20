package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	pb "kitauji/greeter"
	"log"
	"sync"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	serverAddr string
	reqType    string
	reqName    string
)

func init() {
	flag.StringVar(&serverAddr, "addr", "127.0.0.1:55555", "-addr 192.168.1.1:12345")
	flag.StringVar(&reqType, "type", "un", "-type [un|ss|cs|bi]")
	flag.StringVar(&reqName, "name", "", "-name john")
	flag.Parse()
}

func showGrpcError(err error) {
	st, ok := status.FromError(err)
	if !ok {
		log.Printf("Error: %v", err)
		return
	}

	log.Printf("GRPC Error : Code [%d], Message [%s]", st.Code(), st.Message())

	if len(st.Details()) > 0 {
		for _, detail := range st.Details() {
			switch d := detail.(type) {
			case *errdetails.BadRequest:
				log.Printf("  - Details: BadRequest: %v", d)
			case *pb.CustomError:
				log.Printf("  - Details: CustomError: %d, %s", d.ErrorNo, d.Description)
			default:
				log.Printf("  - Details: Unknown: %v", d)
			}
		}
	}
}

func SayHelloUN(cli pb.GreeterClient) {
	req := &pb.HelloRequest{Name: reqName}

	resp, err := cli.SayHelloUN(context.Background(), req)
	if err != nil {
		showGrpcError(err)
		return
	}

	fmt.Printf("Result : %s\n", resp.GetResult())
}

func SayHelloSS(cli pb.GreeterClient) {
	req := &pb.HelloRequest{Name: reqName}

	stream, err := cli.SayHelloSS(context.Background(), req)
	if err != nil {
		log.Printf("SayHelloSS error : %v", err)
		return
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			showGrpcError(err)
			return
		}

		fmt.Printf("Result : %s\n", resp.Result)
	}
}

func SayHelloCS(cli pb.GreeterClient) {
	stream, err := cli.SayHelloCS(context.Background())
	if err != nil {
		log.Printf("SayHelloCS error : %v", err)
		return
	}

	// テスト用に3回だけ送信する
	for i := 0; i < 3; i++ {
		req := &pb.HelloRequest{
			Name: fmt.Sprintf("%s (%d)", reqName, i),
		}

		if err := stream.Send(req); err != nil {
			log.Printf("SayHelloCS: Send() error : %v", err)
			return
		}

		log.Printf("SayHelloCS: Sent a message : %v", req)
		time.Sleep(1 * time.Second)
	}

	stream.CloseSend()
}

func SayHelloBI(cli pb.GreeterClient) {
	log.Printf("SayHelloBI: Start")

	stream, err := cli.SayHelloBI(context.Background())
	if err != nil {
		log.Printf("SayHelloBI error : %v", err)
		return
	}

	wg := &sync.WaitGroup{}

	// 送信用の goroutine
	wg.Add(1)
	go SayHelloBISend(stream, wg)

	// 受信用の goroutine
	wg.Add(1)
	go SayHelloBIRecv(stream, wg)

	wg.Wait()
	log.Printf("SayHelloBI: End")
}

func SayHelloBISend(stream pb.Greeter_SayHelloBIClient, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		req := &pb.HelloRequest{Name: reqName}

		// たまに Name を john にして送る(サーバーから応答が返るため)
		if time.Now().Unix()%5 == 0 {
			req.Name = "john"
		}

		if err := stream.Send(req); err != nil {
			if err == io.EOF {
				log.Printf("SayHelloBI: Client Stream is closed")
			} else {
				showGrpcError(err)
			}
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func SayHelloBIRecv(stream pb.Greeter_SayHelloBIClient, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Printf("SayHelloBI: Server Stream is closed")
			} else {
				showGrpcError(err)
			}
			break
		}

		if st := resp.GetStatus(); st != nil {
			s := status.FromProto(st)
			log.Printf("SayHelloBIRecv: Error. Status Code [%d], Message [%s]", s.Code(), s.Message())

		}

		if resp.GetResult() != "" {
			log.Printf("Result: %v", resp.GetResult())
		}
	}
}

func main() {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial to %s : %v", serverAddr, err)
	}
	defer conn.Close()

	cli := pb.NewGreeterClient(conn)

	switch reqType {
	case "un":
		SayHelloUN(cli)
	case "ss":
		SayHelloSS(cli)
	case "cs":
		SayHelloCS(cli)
	case "bi":
		SayHelloBI(cli)
	default:
		log.Printf("Unknown type.")
	}
}
