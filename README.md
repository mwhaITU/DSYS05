# DSYS05
How to run our program 101

1. Launch two or more clients in two or more different terminals. A unique integer id 
must be specified for each node. We recommend the following:
- go run node.go -name 1
- go run node.go -name 2

2. Launch two or more server in two or more different terminals. A unique port 
must be specified for each node. We recommend the following:
- go run node.go -port 5400
- go run node.go -port 5401

3. Connect the clients to the servers by typing "connect " + the port you wish to connect to. It is very important that you connect to the servers in the same order. If you followed the example above,
you should type the following for each client.
client 1:
- connect 5400
- connect 5401

Client 2:
- connect 5400
- connect 5401

3. You can bid by typing "bid" + an integer. The auction will start when the first bid is made, and lasts 30 seconds. You can get the current highest bid or the result of the auction by typing "result".

4. You can simulate a server crashing by closing the terminal that the servers is running in. When you send the next request, the client will detect that a server has crashed an print an error message, but the requet will still complete.

5. If you want to restart the auction, you'll have to restart all clients and all servers.