package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	gRPC "github.com/DarkLordOfDeadstiny/DSYS-gRPC-template/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type node struct {
	nodeID    int32
	lamport   int32
	nodeSlice []nodeConnection
	mutex     sync.Mutex
	channels  []chan gRPC.Acknowledgement
}
type nodeConnection struct {
	node     gRPC.AuctionClient
	nodeConn *grpc.ClientConn
}

// Same principle as in client. Flags allows for user specific arguments/values
var nodeName = flag.Int("name", 0, "Senders name")

//var port = flag.String("port", "5400", "Listen port")

func main() {
	//parse flag/arguments
	flag.Parse()

	fmt.Println("--- CLIENT APP ---")

	//log to file instead of console
	//setLog()

	//connect to server and close the connection when program closes
	fmt.Println("--- join Server ---")

	node := node{
		nodeID:    int32(*nodeName),
		nodeSlice: make([]nodeConnection, 0),
		lamport:   1,
		channels:  make([]chan gRPC.Acknowledgement, 0),
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
	newChannel := make(chan gRPC.Acknowledgement)

	n.nodeSlice = append(n.nodeSlice, nodeConnection)
	n.channels = append(n.channels, newChannel)
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
			bidString := input[4:]
			bidInt, err := strconv.Atoi(bidString)
			if err != nil {
				fmt.Println("Input a number dipshit")
			} else {
				n.bid(int32(bidInt))
			}
		} else if strings.Contains(input, "result") {
			n.result()
		}
		continue
	}
}

func (n *node) bid(bid int32) {
	result := ""
	//channels := make([]chan gRPC.Acknowledgement, len(n.nodeSlice))
	amount := &gRPC.Amount{
		Lamport: n.lamport,
		Amount:  bid,
		NodeID:  n.nodeID,
	}
	for i, conn := range n.nodeSlice {
		//channels = append(channels, channel)
		go func(index int, connection nodeConnection) {
			fmt.Println("sending bid to index " + strconv.Itoa(index))
			msg, err := connection.node.Bid(context.Background(), amount)
			fmt.Println("bid returned by server")
			if err != nil {
				log.Printf("A server has timed out")
				errorAck := &gRPC.Acknowledgement{
					Status: "ERROR",
				}
				n.channels[index] <- *errorAck
			} else {
				fmt.Println("Trying to insert message into channel")

				n.channels[index] <- *msg
				fmt.Println("Putting message into channel")
			}
		}(i, conn)
	}
	currentHighestBid := 0
	for _, channel := range n.channels {
		response := <-channel
		fmt.Println("Checking repsponse")
		fmt.Println(response.Status)
		if response.Status == "FINISHED" {
			result = "Auction is finished"
			n.finished(response.HighestBid, response.NodeID)
			break
		}
		if response.Status == "EXCEPTION" {
			if currentHighestBid < int(response.HighestBid) {
				currentHighestBid = int(response.HighestBid)
				result = "Bid " + strconv.Itoa(int(bid)) + " not accepted, the highest bid is currently " + strconv.Itoa(currentHighestBid)
			}
		}
		if response.Status == "SUCCESS" {
			if result == "" {
				result = "Bid " + strconv.Itoa(int(bid)) + " is accepted, and is currently the highest bid"
			}
		}

	}
	log.Println(result)
}

func (n *node) result() {
	result := ""
	status := ""
	request := &gRPC.Request{
		Lamport: n.lamport,
	}
	for i, conn := range n.nodeSlice {
		go func(index int, connection nodeConnection) {
			msg, err := connection.node.Result(context.Background(), request)
			if err != nil {
				log.Printf("A server has timed out")
				errorAck := &gRPC.Acknowledgement{
					Status: "ERROR",
				}
				n.channels[index] <- *errorAck
			} else {
				n.channels[index] <- *msg
			}
		}(i, conn)
	}
	currentHighestBid := 0
	winningBidder := 0
	for _, channel := range n.channels {
		response := <-channel
		if response.Status == "FINISHED" {
			status = "FINISHED"
			if currentHighestBid < int(response.HighestBid) {
				currentHighestBid = int(response.HighestBid)
				winningBidder = int(response.NodeID)
				result = "Auction is finished, highest bid was " + strconv.Itoa(int(currentHighestBid)) + " by " + strconv.Itoa(int(winningBidder))
			}
		}
		if response.Status == "ONGOING" {
			if status != "FINISHED" {
				if currentHighestBid < int(response.HighestBid) {
					currentHighestBid = int(response.HighestBid)
					winningBidder = int(response.NodeID)
					result = "Current highest bid is " + strconv.Itoa(int(currentHighestBid)) + " by client" + strconv.Itoa(int(winningBidder))
				}
			}
		}
	}
	log.Printf(result)
}

func (n *node) finished(highestBid int32, nodeID int32) {
	finish := &gRPC.Finish{
		Lamport:    n.lamport,
		NodeID:     nodeID,
		HighestBid: highestBid,
	}
	for index, connection := range n.nodeSlice {
		go func() {
			msg, err := connection.node.Finished(context.Background(), finish)
			if err != nil {
				log.Printf("%s has timed out", connection.node)
				errorAck := &gRPC.Acknowledgement{
					Status: "ERROR",
				}
				n.channels[index] <- *errorAck
			} else {
				n.channels[index] <- *msg
			}
		}()
		for _, channel := range n.channels {
			<-channel
		}
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
