package network

import "bytes"

const (
	protocol      = "tcp" // how about http protocol?
	version       = 1
	commandLength = 12
)

var (
	nodeAddress     string
	minerAddress    string
	KnowNodes       = []string{"localhost: 3000"}
	blocksInTransit = [][]bytes{}
	memoryPool = make(map[string]blockchain.Transaction) 
	// blockchain Transaciont should be imported from our block structure
)

type Addr struct { 
	AddrList [] string
}

type Block struct{
	AddrFrom string

}

type Version struct{
	Version int
	BestHeight int
	AddrFrom string
}

