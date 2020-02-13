package api

import (
	"fmt"
	"time"

	"gitlab.com/sdce/exlib/service"
	pb "gitlab.com/sdce/protogo"
	"google.golang.org/grpc"
)

const (
	apiCallLiveTime = 5 * time.Second
)

//Server serving for rpc api
type Server struct {
	Member  pb.MemberClient
	Trading pb.TradingClient
	//Admin   pb.AdminServiceClient
}

func newMemberClient(memberURL string) (pb.MemberClient, error) {
	// Set up a connection to the server.
	fmt.Println("Member grpc host:" + memberURL)
	conn, err := grpc.Dial(memberURL, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return pb.NewMemberClient(conn), nil
}

func newTradingClient(tradingURL string) (pb.TradingClient, error) {
	// Set up a connection to the server.
	fmt.Println("Trading grpc host:" + tradingURL)
	conn, err := grpc.Dial(tradingURL, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return pb.NewTradingClient(conn), nil
}

func newAdminClient(adminURL string) (pb.AdminServiceClient, error) {
	// Set up a connection to the server.
	fmt.Println("Admin grpc host:" + adminURL)
	conn, err := grpc.Dial(adminURL, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return pb.NewAdminServiceClient(conn), nil
}

//New api server
func New(config *service.Config) (api *Server, err error) {
	member, err := newMemberClient(config.Member)
	if err != nil {
		return nil, err
	}
	trading, err := newTradingClient(config.Trading)
	if err != nil {
		return nil, err
	}
	// admin, err := newAdminClient(config.Admin)
	// if err != nil {
	// 	return nil, err
	// }

	return &Server{
		Member:  member,
		Trading: trading,
		//Admin:   admin,
	}, nil
}
