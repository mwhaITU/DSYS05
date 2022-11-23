package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	// this has to be the same as the go.mod module,
	// followed by the path to the folder the proto file is in.
	"github.com/DarkLordOfDeadstiny/DSYS-gRPC-template/proto"
	gRPC "github.com/DarkLordOfDeadstiny/DSYS-gRPC-template/proto"

	"google.golang.org/grpc"
)

type Server struct {
	gRPC.UnimplementedAuctionServer        // You need this line if you have a server
	name                            string // Not required but useful if you want to name your server
	port                            string // Not required but useful if your server needs to know what port it's listening to
	channel                         map[int32]chan *proto.Message
	lamportClock                    int32
	highestBid                      int32
	winningBidder                   int32
	mutex                           sync.Mutex
	finished                        bool
}

// flags are used to get arguments from the terminal. Flags take a value, a default value and a description of the flag.
// to use a flag then just add it as an argument when running the program.
var serverName = flag.String("name", "default", "Senders name") // set with "-name <name>" in terminal
var port = flag.String("port", "5400", "Server port")           // set with "-port <port>" in terminal

func main() {
	// This parses the flags and sets the correct/given corresponding values.
	flag.Parse()
	fmt.Println("Server is starting...")

	// starts a goroutine executing the launchServer method.
	go launchServer()

	// This makes sure that the main method is "kept alive"/keeps running
	for {
		time.Sleep(time.Second * 5)
	}
}

func launchServer() {
	log.Printf("Server %s: Attempts to create listener on port %s\n", *serverName, *port)

	// Create listener tcp on given port or default port 5400
	// Insert your device's IP before the colon in the print statement
	list, err := net.Listen("tcp", "localhost:"+*port)
	if err != nil {
		log.Printf("Server %s: Failed to listen on port %s: %v", *serverName, *port, err) //If it fails to listen on the port, run launchServer method again with the next value/port in ports array
		return
	}

	// makes gRPC server using the options
	// you can add options here if you want or remove the options part entirely
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	// makes a new server instance using the name and port from the flags.
	server := &Server{
		name:         *serverName,
		port:         *port,
		lamportClock: 0,
		channel:      make(map[int32]chan *proto.Message),
	}

	gRPC.RegisterAuctionServer(grpcServer, server) //Registers the server to the gRPC server

	log.Printf("Server %s: Listening on port %s\n", *serverName, *port)
	fmt.Println("End of method was reached")
	go server.startAuction()
	if err := grpcServer.Serve(list); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
	for {
		time.Sleep(time.Second * 5)
	}
}

func (s *Server) startAuction() {
	fmt.Println("Auction is waiting to start")
	s.finished = false
	for {
		if s.highestBid > 0 {
			fmt.Println("Auction started")
			break
		}
	}
	time.Sleep(30 * time.Second)
	fmt.Println("Auction done")
	s.finished = true
}

func (s *Server) Bid(ctx context.Context, amount *gRPC.Amount) (*gRPC.Acknowledgement, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.highestBid == 0 {

	}

	if amount.Lamport > s.lamportClock {
		s.lamportClock = amount.Lamport
	}
	log.Printf("Received bid amount: %v \n", amount.Amount)
	s.lamportClock++
	if s.finished {
		return &proto.Acknowledgement{Status: "FINISHED", Lamport: s.lamportClock, HighestBid: s.highestBid, NodeID: s.winningBidder}, nil
	}
	if amount.Amount > s.highestBid {
		fmt.Println("amount was higher than highest bid")
		s.highestBid = amount.Amount
		s.winningBidder = amount.NodeID
		return &proto.Acknowledgement{Status: "SUCCESS", Lamport: s.lamportClock, HighestBid: s.highestBid}, nil
	}
	return &proto.Acknowledgement{Status: "EXCEPTION", Lamport: s.lamportClock, HighestBid: s.highestBid}, nil
}

func (s *Server) Result(ctx context.Context, request *gRPC.Request) (*gRPC.Acknowledgement, error) {
	if request.Lamport > s.lamportClock {
		s.lamportClock = request.Lamport
	}
	s.lamportClock++
	if s.finished {
		return &gRPC.Acknowledgement{HighestBid: s.highestBid, Lamport: s.lamportClock, NodeID: s.winningBidder, Status: "FINISHED"}, nil
	}
	return &gRPC.Acknowledgement{HighestBid: s.highestBid, Lamport: s.lamportClock, NodeID: s.winningBidder, Status: "ONGOING"}, nil
}

func (s *Server) Finished(ctx context.Context, request *gRPC.Finish) (*gRPC.Acknowledgement, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.finished {
		s.finished = true
		s.highestBid = request.HighestBid
		s.winningBidder = request.NodeID
	}
	if request.Lamport > s.lamportClock {
		s.lamportClock = request.Lamport
	}
	s.lamportClock++
	return &gRPC.Acknowledgement{HighestBid: s.highestBid, Lamport: s.lamportClock}, nil
}
