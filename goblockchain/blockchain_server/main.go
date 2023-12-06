package main

import (
	"flag"
	"log"
)

func init() {
	log.SetPrefix("Blockchain: ")
}

func main() {
	port := flag.Uint("port", 5000, "TCP Port Number for Blockchain Server")
	flag.Parse()
	app := NewBlockchainServer(uint16(*port))
	app.Run()
}

//ValidProof uses nonce, previousHash, transactions and checks if it is real
//When mining, it takes the transaction pool, LastBlock.currentHash, and generates needed nonce, so that when we pass these three arguments
//However, when we pass them again, it does not work. Why?

//How do we change the difficulty? When mining, if in the next 20 seconds, the Mining portion still contains some of the previous transactions,
//then we need to lower the difficulty. For that, we need to track the transaction pool.
