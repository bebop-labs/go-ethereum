package catalyst

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/eth"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/scroll-tech/go-ethereum/trie"
)

func RegisterL2Engine(stack *node.Node, backend *eth.Ethereum) error {
	chainconfig := backend.BlockChain().Config()
	if chainconfig.TerminalTotalDifficulty == nil {
		return errors.New("catalyst started without valid total difficulty")
	}

	stack.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Version:       "1.0",
			Service:       newL2ConsensusAPI(backend),
			Public:        true,
			Authenticated: true,
		},
	})
	return nil
}

type l2ConsensusAPI struct {
	eth      *eth.Ethereum
	verified map[common.Hash]executionResult // stored execution result of the next block that to be committed
}

func newL2ConsensusAPI(eth *eth.Ethereum) *l2ConsensusAPI {
	return &l2ConsensusAPI{
		eth:      eth,
		verified: make(map[common.Hash]executionResult),
	}
}

type executionResult struct {
	block    *types.Block
	state    *state.StateDB
	receipts types.Receipts
	procTime time.Duration
}

func (api *l2ConsensusAPI) AssembleL2Block(params AssembleL2BlockParams) (*ExecutableL2Data, error) {
	log.Info("Producing block", "block number", params.Number)
	parent := api.eth.BlockChain().CurrentHeader()
	expectedBlockNumber := parent.Number.Uint64() + 1
	if params.Number != expectedBlockNumber {
		log.Warn("Cannot assemble block with discontinuous block number", "expected number", expectedBlockNumber, "actual number", params.Number)
		return nil, fmt.Errorf("cannot assemble block with discontinuous block number %d, expected number is %d", params.Number, expectedBlockNumber)
	}
	transactions := make(types.Transactions, 0, len(params.Transactions))
	for i, otx := range params.Transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(otx); err != nil {
			return nil, fmt.Errorf("transaction %d is not valid: %v", i, err)
		}
		transactions = append(transactions, &tx)
	}

	start := time.Now()
	block, state, receipts, err := api.eth.Miner().GetSealingBlockAndState(parent.Hash(), time.Now(), transactions)
	if err != nil {
		return nil, err
	}

	// Do not produce new block if no transaction is involved
	if block.TxHash() == types.EmptyRootHash {
		return nil, nil
	}
	api.verified[block.Hash()] = executionResult{
		block:    block,
		state:    state,
		receipts: receipts,
		procTime: time.Since(start),
	}
	return &ExecutableL2Data{
		ParentHash:   block.ParentHash(),
		Number:       block.NumberU64(),
		Miner:        block.Coinbase(),
		Timestamp:    block.Time(),
		GasLimit:     block.GasLimit(),
		BaseFee:      block.BaseFee(),
		Transactions: encodeTransactions(block.Transactions()),

		StateRoot:   block.Root(),
		GasUsed:     block.GasUsed(),
		ReceiptRoot: block.ReceiptHash(),
		LogsBloom:   block.Bloom().Bytes(),
	}, nil
}

func (api *l2ConsensusAPI) ValidateL2Block(params ExecutableL2Data) (*GenericResponse, error) {
	parent := api.eth.BlockChain().CurrentBlock()
	expectedBlockNumber := parent.NumberU64() + 1
	if params.Number != expectedBlockNumber {
		log.Warn("Cannot assemble block with discontinuous block number", "expected number", expectedBlockNumber, "actual number", params.Number)
		return nil, fmt.Errorf("cannot assemble block with discontinuous block number %d, expected number is %d", params.Number, expectedBlockNumber)
	}
	if params.ParentHash != parent.Hash() {
		log.Warn("Wrong parent hash", "expected block hash", parent.TxHash().Hex(), "actual block hash", params.ParentHash.Hex())
		return nil, fmt.Errorf("wrong parent hash: %s, expected parent hash is %s", params.ParentHash, parent.Hash())
	}

	block, err := api.paramsToBlock(params, types.BLSData{})
	if err != nil {
		return nil, err
	}
	_, verified := api.verified[block.Hash()]
	if verified {
		return &GenericResponse{
			true,
		}, nil
	}

	if err := api.VerifyBlock(block); err != nil {
		return &GenericResponse{
			false,
		}, nil
	}

	if err := api.eth.BlockChain().Validator().ValidateBody(block); err != nil {
		log.Error("error validating body", "error", err)
		return &GenericResponse{
			false,
		}, nil
	}

	stateDB, receipts, procTime, err := api.eth.BlockChain().ProcessBlock(block, parent.Header())
	if err != nil {
		log.Error("error processing block", "error", err)
		return &GenericResponse{
			false,
		}, nil
	}

	api.verified[block.Hash()] = executionResult{
		block:    block,
		state:    stateDB,
		receipts: receipts,
		procTime: procTime,
	}
	return &GenericResponse{
		true,
	}, nil
}

func (api *l2ConsensusAPI) NewL2Block(params ExecutableL2Data, bls types.BLSData) (err error) {
	parent := api.eth.BlockChain().CurrentBlock()
	expectedBlockNumber := parent.NumberU64() + 1
	if params.Number != expectedBlockNumber {
		log.Warn("Cannot assemble block with discontinuous block number", "expected number", expectedBlockNumber, "actual number", params.Number)
		return fmt.Errorf("cannot assemble block with discontinuous block number %d, expected number is %d", params.Number, expectedBlockNumber)
	}
	if params.ParentHash != parent.Hash() {
		log.Warn("Wrong parent hash", "expected block hash", parent.Hash().Hex(), "actual block hash", params.ParentHash.Hex())
		return fmt.Errorf("wrong parent hash: %s, expected parent hash is %s", params.ParentHash, parent.Hash())
	}

	block, err := api.paramsToBlock(params, bls)
	if err != nil {
		return err
	}

	bas, verified := api.verified[block.Hash()]
	if verified {
		err = api.eth.BlockChain().WriteStateAndSetHead(block, bas.receipts, bas.state, bas.procTime)
		if err == nil {
			api.verified = make(map[common.Hash]executionResult)
		}
		return err
	}

	if err := api.VerifyBlock(block); err != nil {
		log.Error("failed to verify block", "error", err)
		return err
	}

	stateDB, receipts, procTime, err := api.eth.BlockChain().ProcessBlock(block, parent.Header())
	if err != nil {
		return err
	}
	return api.eth.BlockChain().WriteStateAndSetHead(block, receipts, stateDB, procTime)
}

func (api *l2ConsensusAPI) paramsToBlock(params ExecutableL2Data, blsData types.BLSData) (*types.Block, error) {
	header := &types.Header{
		ParentHash: params.ParentHash,
		Number:     big.NewInt(int64(params.Number)),
		GasUsed:    params.GasUsed,
		GasLimit:   params.GasLimit,
		Time:       params.Timestamp,
		Coinbase:   params.Miner,
		Extra:      params.Extra,
		BLSData:    blsData,
		BaseFee:    params.BaseFee,
	}
	api.eth.Engine().Prepare(api.eth.BlockChain(), header)

	txs, err := decodeTransactions(params.Transactions)
	if err != nil {
		return nil, err
	}
	header.TxHash = types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil))
	header.ReceiptHash = params.ReceiptRoot
	header.Root = params.StateRoot
	header.Bloom = types.BytesToBloom(params.LogsBloom)
	return types.NewBlockWithHeader(header).WithBody(txs, nil), nil
}

func (api *l2ConsensusAPI) VerifyBlock(block *types.Block) error {
	if err := api.eth.Engine().VerifyHeader(api.eth.BlockChain(), block.Header(), false); err != nil {
		log.Warn("failed to verify header", "error", err)
		return err
	}
	if !api.eth.BlockChain().Config().Scroll.IsValidTxCount(len(block.Transactions())) {
		return errors.New("invalid tx count")
	}
	return nil
}
