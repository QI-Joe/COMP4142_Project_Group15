package block

import (
	"awesomeProject/utils"
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	MAX_MINING_DIFFICULTY             = 8
	DIFFICULTY_ADJUSTMENT_TIME        = 20
	MIN_MINING_DIFFICULTY             = 4
	MINING_SENDER                     = "THE BLOCKCHAIN"
	MINING_REWARD                     = 1.0
	MINING_TIMER_SEC                  = 20
	BLOCKCHAIN_PORT_RANGE_START       = 5000
	BLOCKCHAIN_PORT_RANGE_END         = 5003
	BLOCKCHAIN_NEIGHBOR_SYNC_TIME_SEC = 10
	NEIGHBOR_IP_RANGE_START           = 0
	NEIGHBOR_IP_RANGE_END             = 0
)

var DynamicDifficulty = MIN_MINING_DIFFICULTY
var LastChainLength int

type Block struct { //here we define the structure of the Block itself
	nonce        int
	previousHash [32]byte
	currentHash  [32]byte
	timestamp    int64
	transactions []*Transaction
	difficulty   int
	index        int
	merkleRoot   [32]byte
}

func (b *Block) PreviousHash() [32]byte {
	return b.previousHash
}

func (b *Block) Nonce() int {
	return b.nonce
}

func (b *Block) Transactions() []*Transaction {
	return b.transactions
}

func (bc *Blockchain) ValidChain(chain []*Block) bool {
	preBlock := chain[0]
	currentIndex := 1
	for currentIndex < len(chain) {
		b := chain[currentIndex]
		if b.previousHash != preBlock.currentHash {
			log.Println("ERROR: PREVIOUS HASH IS FALSE")
			return false
		}

		log.Println("Nonce: ", b.Nonce())
		log.Println("Previous Hash: ", b.PreviousHash())
		log.Println("Transactions: ", b.Transactions())

		if !bc.ValidProof(b.Nonce(), b.PreviousHash(), b.Transactions(), b.difficulty) {
			log.Println("ERROR: NONCE IS FALSE")
			return false
		}

		if b.merkleRoot != GenerateMerkleTree(b.transactions) {
			log.Println("ERROR: TRANSACTIONS ARE TAMPERED")
			return false
		}

		preBlock = b
		currentIndex += 1
	}
	return true
}

func (bc *Blockchain) Chain() []*Block {
	return bc.chain
}

func (bc *Blockchain) ResolveConflicts() bool {
	var longestChain []*Block = nil
	maxLength := len(bc.chain)

	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/chain", n)
		resp, _ := http.Get(endpoint)
		if resp.StatusCode == 200 {
			var bcResp Blockchain
			decoder := json.NewDecoder(resp.Body)
			_ = decoder.Decode(&bcResp)

			chain := bcResp.Chain()
			bcResp.Print()

			log.Println(bc.ValidChain(chain))
			log.Println(len(chain))
			if len(chain) > maxLength && bc.ValidChain(chain) {
				maxLength = len(chain)
				longestChain = chain
			}
		}
	}

	if longestChain != nil {
		bc.chain = longestChain
		log.Printf("Resovle confilicts replaced")
		return true
	}
	log.Printf("Resovle conflicts not replaced")
	return false
}

func (b *Block) Print() {
	fmt.Printf("timestamp        %d\n", b.timestamp)
	fmt.Printf("nonce            %d\n", b.nonce)
	fmt.Printf("previous_hash    %x\n", b.previousHash)
	fmt.Printf("current_hash     %x\n", b.currentHash)
	for _, t := range b.transactions {
		t.Print()
	}
}

func NewBlock(nonce int, previousHash [32]byte, transactions []*Transaction, difficulty int) *Block {
	b := new(Block)
	b.timestamp = time.Now().UnixNano()
	b.nonce = nonce
	b.previousHash = previousHash
	b.transactions = transactions
	b.difficulty = difficulty
	return b
}

func (b *Block) AddCurrentHash() {
	b.currentHash = b.Hash()
}

type Blockchain struct {
	transactionPool   []*Transaction
	chain             []*Block
	blockchainAddress string
	port              uint16
	mux               sync.Mutex

	neighbors    []string
	muxNeighbors sync.Mutex

	muxDifficulty sync.Mutex
}

func NewBlockchain(blockchainAddress string, port uint16) *Blockchain {
	bc := new(Blockchain)
	b := &Block{}
	firstBlock := NewBlock(0, b.Hash(), []*Transaction{}, DynamicDifficulty)
	//bc.CreateBlock(0, b.Hash())
	bc.chain = append(bc.chain, firstBlock)
	bc.transactionPool = []*Transaction{}
	bc.blockchainAddress = blockchainAddress
	bc.port = port
	bc.chain[0].AddCurrentHash()
	bc.chain[0].previousHash = bc.chain[0].currentHash
	bc.chain[0].index = 1
	LastChainLength = 1
	return bc
}

func (bc *Blockchain) Run() {
	bc.StartSyncNeighbors()
	bc.ResolveConflicts()
	bc.StartSyncDifficulty()
}

func (bc *Blockchain) SetNeighbors() {
	bc.neighbors = utils.FindNeighbors(
		"127.0.0.1", bc.port,
		NEIGHBOR_IP_RANGE_START, NEIGHBOR_IP_RANGE_END,
		BLOCKCHAIN_PORT_RANGE_START, BLOCKCHAIN_PORT_RANGE_END)
	log.Printf("%v", bc.neighbors)
}

func (bc *Blockchain) SyncNeighbors() {
	bc.muxNeighbors.Lock()
	defer bc.muxNeighbors.Unlock()
	bc.SetNeighbors()
}

func (bc *Blockchain) StartSyncNeighbors() {
	bc.SyncNeighbors()
	_ = time.AfterFunc(time.Second*BLOCKCHAIN_NEIGHBOR_SYNC_TIME_SEC, bc.StartSyncNeighbors)
}

func (bc *Blockchain) TransactionPool() []*Transaction {
	return bc.transactionPool
}

func (bc *Blockchain) ClearTransactionPool() {
	bc.transactionPool = bc.transactionPool[:0]
}

func (bc *Blockchain) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Blocks []*Block `json:"chain"`
	}{
		Blocks: bc.chain,
	})
}

func (bc *Blockchain) UnmarshalJSON(data []byte) error {
	v := &struct {
		Blocks *[]*Block `json:"chain"`
	}{
		Blocks: &bc.chain,
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	return nil
}

func (bc *Blockchain) CreateBlock(nonce int, previousHash [32]byte, difficulty int) *Block {
	b := NewBlock(nonce, previousHash, bc.TransactionPool(), difficulty)
	tb := &Block{nonce: nonce, previousHash: previousHash, transactions: bc.TransactionPool()}
	b.currentHash = tb.Hash()
	b.index = bc.LastBlock().index + 1
	b.merkleRoot = GenerateMerkleTree(bc.TransactionPool())
	bc.chain = append(bc.chain, b)
	bc.transactionPool = []*Transaction{}

	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/transactions", n)
		client := &http.Client{}
		req, _ := http.NewRequest("DELETE", endpoint, nil)
		resp, _ := client.Do(req)
		log.Printf("%v", resp)
	}
	return b
}

func GenerateMerkleTree(transactions []*Transaction) [32]byte {
	hashOfTransactions := make([][32]byte, len(transactions))
	for i, _ := range transactions {
		t, _ := transactions[i].MarshalJSON()
		hashOfTransactions[i] = sha256.Sum256(t)
	}
	log.Println(len(hashOfTransactions))
	pointer := 0
	placer := 0
	lim := len(hashOfTransactions) - 1
	for lim > 0 {
		for pointer+1 <= lim {
			log.Println("Pointer: ", pointer)
			log.Println("Placer: ", placer)
			a := string(hashOfTransactions[pointer][:])
			b := string(hashOfTransactions[pointer+1][:])
			c := a + b
			hashOfTransactions[placer] = sha256.Sum256([]byte(c))
			placer++
			pointer += 2
			if pointer == lim {
				hashOfTransactions[placer] = hashOfTransactions[pointer]
			}
			log.Println("Placed hash: ", string(hashOfTransactions[placer][:]))
		}
		pointer = 0
		placer = 0
		log.Println("LIM: ", lim)
		lim = int(math.Floor(float64(lim / 2)))
		log.Println("LIM: ", lim)
	}
	log.Println(hashOfTransactions[0])
	return hashOfTransactions[0]
}

func (bc *Blockchain) Print() {
	fmt.Printf("%s\n", strings.Repeat("*", 50))
	for i, block := range bc.chain {
		fmt.Printf("%s Chain %d %s\n", strings.Repeat("=", 25), i, strings.Repeat("=", 25))
		block.Print()
	}
	fmt.Printf("%s\n", strings.Repeat("*", 50))
}

func (b *Block) Hash() [32]byte {
	m, _ := json.Marshal(b)
	//fmt.Println(string(m))
	return sha256.Sum256([]byte(m))
}

func (b *Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Timestamp    int64          `json:"timestamp"`
		Nonce        int            `json:"nonce"`
		PreviousHash string         `json:"previousHash"`
		CurrentHash  string         `json:"currentHash"`
		Transactions []*Transaction `json:"transactions"`
		Difficulty   int            `json:"difficulty"`
		Index        int            `json:"index"`
		MerkleRoot   string         `json:"merkleRoot"`
	}{
		Timestamp:    b.timestamp,
		Nonce:        b.nonce,
		PreviousHash: fmt.Sprintf("%x", b.previousHash),
		CurrentHash:  fmt.Sprintf("%x", b.currentHash),
		Transactions: b.transactions,
		Difficulty:   b.difficulty,
		Index:        b.index,
		MerkleRoot:   fmt.Sprintf("%x", b.merkleRoot),
	})
}

func (b *Block) UnmarshalJSON(data []byte) error {
	var previousHash string
	var currentHash string
	var merkleRoot string
	v := &struct {
		Timestamp    *int64          `json:"timestamp"`
		Nonce        *int            `json:"nonce"`
		PreviousHash *string         `json:"previousHash"`
		CurrentHash  *string         `json:"currentHash"`
		Transactions *[]*Transaction `json:"transactions"`
		Difficulty   *int            `json:"difficulty"`
		Index        *int            `json:"index"`
		MerkleRoot   *string         `json:"merkleRoot"`
	}{
		Timestamp:    &b.timestamp,
		Nonce:        &b.nonce,
		PreviousHash: &previousHash,
		CurrentHash:  &currentHash,
		Transactions: &b.transactions,
		Difficulty:   &b.difficulty,
		Index:        &b.index,
		MerkleRoot:   &merkleRoot,
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	ph, _ := hex.DecodeString(*v.PreviousHash)
	copy(b.previousHash[:], ph[:32])
	ch, _ := hex.DecodeString(*v.CurrentHash)
	copy(b.currentHash[:], ch[:32])
	mr, _ := hex.DecodeString(*v.MerkleRoot)
	copy(b.merkleRoot[:], mr[:32])
	return nil
}

func (bc *Blockchain) LastBlock() *Block {
	return bc.chain[len(bc.chain)-1]
}

type Transaction struct {
	senderBlockchainAddress    string
	recipientBlockchainAddress string
	value                      float32
}

func NewTransaction(sender string, recipient string, value float32) *Transaction {
	return &Transaction{sender, recipient, value}
}

func (t *Transaction) Print() {
	fmt.Printf("%s\n", strings.Repeat("-", 40))
	fmt.Printf("  sender_blockchain_adress      %s\n", t.senderBlockchainAddress)
	fmt.Printf("  recipient_blockchain_adress   %s\n", t.recipientBlockchainAddress)
	fmt.Printf("  value                         %0.1f\n", t.value)
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sender    string  `json:"sender_blockchain_address"`
		Recipient string  `json:"recipient_blockchain_address"`
		Value     float32 `json:"value"`
	}{
		Sender:    t.senderBlockchainAddress,
		Recipient: t.recipientBlockchainAddress,
		Value:     t.value,
	})
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	v := &struct {
		Sender    *string  `json:"sender_blockchain_address"`
		Recipient *string  `json:"recipient_blockchain_address"`
		Value     *float32 `json:"value"`
	}{
		Sender:    &t.senderBlockchainAddress,
		Recipient: &t.recipientBlockchainAddress,
		Value:     &t.value,
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	return nil
}

func (bc *Blockchain) CreateTransaction(sender string, recipient string, value float32,
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	isTransacted := bc.AddTransaction(sender, recipient, value, senderPublicKey, s)

	if isTransacted {
		for _, n := range bc.neighbors {
			publicKeyStr := fmt.Sprintf("%064x%064x", senderPublicKey.X.Bytes(),
				senderPublicKey.Y.Bytes())
			signatureStr := s.String()
			bt := &TransactionRequest{
				&sender, &recipient, &publicKeyStr, &value, &signatureStr}
			m, _ := json.Marshal(bt)
			buf := bytes.NewBuffer(m)
			endpoint := fmt.Sprintf("http://%s/transactions", n)
			client := &http.Client{}
			req, _ := http.NewRequest("PUT", endpoint, buf)
			resp, _ := client.Do(req)
			log.Printf("%v", resp)
		}
	}

	return isTransacted
}

func (bc *Blockchain) AddTransaction(sender string, recipient string, value float32,
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	t := NewTransaction(sender, recipient, value)

	if sender == MINING_SENDER {
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	}

	if bc.VerifyTransactionSignature(senderPublicKey, s, t) {
		/*
			if bc.CalculateTotalAmount(sender) < value {
				log.Println("ERROR: Not enough balance in a wallet")
				return false
			}
		*/
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	} else {
		log.Println("ERROR: Verify Transaction")
	}
	return false

}

func (bc *Blockchain) VerifyTransactionSignature(senderPublicKey *ecdsa.PublicKey, s *utils.Signature, t *Transaction) bool {
	m, _ := json.Marshal(t)
	h := sha256.Sum256([]byte(m))
	return ecdsa.Verify(senderPublicKey, h[:], s.R, s.S)
}

func (bc *Blockchain) CopyTransactionPool() []*Transaction {
	transactions := make([]*Transaction, 0)
	for _, t := range bc.transactionPool {
		transactions = append(transactions, NewTransaction(t.senderBlockchainAddress, t.recipientBlockchainAddress, t.value))
	}
	return transactions
}

func (bc *Blockchain) ValidProof(nonce int, previousHash [32]byte, transactions []*Transaction, difficulty int) bool {
	zeros := strings.Repeat("0", difficulty)
	guessBlock := Block{nonce: nonce, previousHash: previousHash, transactions: transactions}
	guessHash := fmt.Sprintf("%x", guessBlock.Hash())
	//log.Println(guessHash)
	return zeros == guessHash[:difficulty]
}

func (bc *Blockchain) ProofOfWork() int {
	nonce := 0
	transactions := bc.CopyTransactionPool()
	for !bc.ValidProof(nonce, bc.LastBlock().currentHash, transactions, DynamicDifficulty) {
		nonce += 1
	}
	return nonce
}

func (bc *Blockchain) AdjustDifficulty() int {
	difficulty := bc.LastBlock().difficulty
	if len(bc.chain)-LastChainLength > 1 {
		difficulty++
	} else if len(bc.chain)-LastChainLength < 1 {
		difficulty--
	}
	if difficulty < MIN_MINING_DIFFICULTY {
		difficulty = 4
	} else if difficulty > MAX_MINING_DIFFICULTY {
		difficulty = 8
	}
	LastChainLength = len(bc.chain)
	log.Printf("Difficulty adjusted to - %v\n", difficulty)
	return difficulty
}

func (bc *Blockchain) SyncDifficulty() {
	bc.muxDifficulty.Lock()
	defer bc.muxDifficulty.Unlock()
	DynamicDifficulty = bc.AdjustDifficulty()
}

func (bc *Blockchain) StartSyncDifficulty() {
	bc.SyncDifficulty()
	_ = time.AfterFunc(time.Second*DIFFICULTY_ADJUSTMENT_TIME, bc.StartSyncDifficulty)
}

func (bc *Blockchain) Mining() bool {
	bc.mux.Lock()
	defer bc.mux.Unlock()

	if len(bc.transactionPool) == 0 {
		return false
	}

	bc.AddTransaction(MINING_SENDER, bc.blockchainAddress, MINING_REWARD, nil, nil)
	nonce := bc.ProofOfWork()
	previousHash := bc.LastBlock().currentHash
	bc.CreateBlock(nonce, previousHash, DynamicDifficulty)
	log.Println("action=mining, status=success")
	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/consensus", n)
		client := &http.Client{}
		req, _ := http.NewRequest("PUT", endpoint, nil)
		resp, _ := client.Do(req)
		log.Printf("%v", resp)
	}
	return true
}

func (bc *Blockchain) StartMining() {
	bc.Mining()
	_ = time.AfterFunc(time.Second*MINING_TIMER_SEC, bc.StartMining)
}

func (bc *Blockchain) CalculateTotalAmount(blockchainAddress string) float32 {
	var totalAmount float32 = 0.0
	for _, b := range bc.chain {
		for _, t := range b.transactions {
			value := t.value
			if blockchainAddress == t.recipientBlockchainAddress {
				totalAmount += value
			}
			if blockchainAddress == t.senderBlockchainAddress {
				totalAmount -= value
			}
		}
	}
	return totalAmount
}

type TransactionRequest struct {
	SenderBlockchainAddress    *string  `json:"sender_blockchain_address"`
	RecipientBlockchainAddress *string  `json:"recipient_blockchain_address"`
	SenderPublicKey            *string  `json:"sender_public_key"`
	Value                      *float32 `json:"value"`
	Signature                  *string  `json:"signature"`
}

func (tr *TransactionRequest) Validate() bool {
	if tr.Signature == nil || tr.RecipientBlockchainAddress == nil || tr.SenderBlockchainAddress == nil || tr.SenderPublicKey == nil || tr.Value == nil {
		return false
	}
	return true
}

type AmountResponse struct {
	Amount float32 `json:"amount"`
}

func (ar *AmountResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount float32 `json:"amount"`
	}{
		Amount: ar.Amount,
	})
}
