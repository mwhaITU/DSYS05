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
	channel                         map[string]chan *proto.Message
	lamportClock                    int32
	highestBid                      int32
	mutex                           sync.Mutex
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
	list, err := net.Listen("tcp", "localhost:5400")
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
		channel:      make(map[string]chan *proto.Message),
	}

	gRPC.RegisterAuctionServer(grpcServer, server) //Registers the server to the gRPC server

	log.Printf("Server %s: Listening on port %s\n", *serverName, *port)

	if err := grpcServer.Serve(list); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
}

func (s *Server) JoinAuction(msg *proto.Message, msgStream proto.Auction_JoinAuctionServer) error {

	msgChannel := make(chan *proto.Message)
	s.channel[msg.Sender] = msgChannel
	for {
		select {
		case <-msgStream.Context().Done():
			return nil
		case msg := <-msgChannel:
			s.lamportClock++
			msg.Lamport = s.lamportClock
			msgStream.Send(msg)
		}
	}
}

func (s *Server) Bid(ctx context.Context, amount *gRPC.Amount) (*gRPC.Acknowledgement, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if amount.Lamport > s.lamportClock {
		s.lamportClock = amount.Lamport
	}
	log.Printf("Received bid amount: %v \n", amount.Amount)
	s.lamportClock++
	if amount.Amount > s.highestBid {
		s.highestBid = amount.Amount
		return &proto.Acknowledgement{Status: "ACCEPTED", Lamport: s.lamportClock}, nil
	}
	return &proto.Acknowledgement{Status: "ERROR", Lamport: s.lamportClock}, nil
}

func (s *Server) Result(ctx context.Context, request *gRPC.Request) (*gRPC.Amount, error) {
	if request.Lamport > s.lamportClock {
		s.lamportClock = request.Lamport
	}
	s.lamportClock++
	return &gRPC.Amount{Amount: s.highestBid, Lamport: s.lamportClock}, nil
}
