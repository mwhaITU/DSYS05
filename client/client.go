package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/DarkLordOfDeadstiny/DSYS-gRPC-template/proto"
	gRPC "github.com/DarkLordOfDeadstiny/DSYS-gRPC-template/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type node struct {
	listenPort string
	nodeID     int32
	//channels        map[string]chan gRPC.Reply
	lamport         int32
	nodeSlice       []nodeConnection
	mutex           sync.Mutex
	state           string
	requests        []string
	lamportRequest  int32
	repliesReceived chan int
}
type nodeConnection struct {
	node     gRPC.AuctionClient
	nodeConn *grpc.ClientConn
}

// Same principle as in client. Flags allows for user specific arguments/values
var nodeName = flag.Int("name", 0, "Senders name")
var port = flag.String("port", "5400", "Listen port")

func main() {
	//parse flag/arguments
	flag.Parse()

	fmt.Println("--- CLIENT APP ---")

	//log to file instead of console
	//setLog()

	//connect to server and close the connection when program closes
	fmt.Println("--- join Server ---")

	node := node{
		listenPort:      *port,
		nodeID:          int32(*nodeName),
		nodeSlice:       make([]nodeConnection, 0),
		lamport:         1,
		repliesReceived: make(chan int),
	}
	go node.parseInput()
	for {
		time.Sleep(5 * time.Second)
	}
}

func (n *node) ConnectToNode(port string) {
	//dial options
	//the server is not using TLS, so we use insecure credentials
	//(should be fine for local testing but not in the real world)
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))

	//use context for timeout on the connection
	timeContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() //cancel the connection when we are done

	//dial the server to get a connection to it
	log.Printf("client %v: Attempts to dial on port %v\n", n.nodeID, port)
	// Insert your device's IP before the colon in the print statement
	conn, err := grpc.DialContext(timeContext, fmt.Sprintf(":%s", port), opts...)
	if err != nil {
		log.Printf("Fail to Dial : %v", err)
		return
	}

	// makes a client from the server connection and saves the connection
	// and prints rather or not the connection was is READY
	nodeConnection := nodeConnection{
		node:     gRPC.NewAuctionClient(conn),
		nodeConn: conn,
	}
	n.nodeSlice = append(n.nodeSlice, nodeConnection)
	log.Println("the connection is: ", conn.GetState().String())
}

func (n *node) parseInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--------------------")

	//Infinite loop to listen for clients input.
	for {
		fmt.Print("-> ")

		//Read input into var input and any errors into err
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		input = strings.TrimSpace(input) //Trim input
		if strings.Contains(input, "connect") {
			portString := input[8:12]
			if err != nil {
				// ... handle error
				panic(err)
			}
			n.ConnectToNode(portString)
		} else if strings.Contains(input, "bid") {

		} else if strings.Contains(input, "result") {

		}
		continue
	}
}

func (n *node) joinAuction(ctx context.Context, server proto.AuctionClient) {
	ack := proto.Message{Sender: n.nodeID}
	stream, err := server.JoinAuction(ctx, &ack)
	if err != nil {
		log.Fatalf("client.JoinChat(ctx, &channel) throws: %v", err)
	}
	fmt.Printf("Joined auction: %v \n", n.nodeID)
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				fmt.Println("not working")
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive message from channel joining. \nErr: %v", err)
			}
			fmt.Println("--------------------")
		}
	}()

	<-waitc
}

func (n *node) bid(bid int32) {
	for _, element := range n.nodeSlice {
		go n.sendBid(element, bid)
	}
}

func (n *node) sendBid(connection nodeConnection, bid int32) {
	msg, err := connection.node.Bid(context.Background(), bid)
	if err != nil {
		log.Printf("Cannot send message: error: %v", err)
	}
}

// sets the logger to use a log.txt file instead of the console
func setLog() {
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)
}
