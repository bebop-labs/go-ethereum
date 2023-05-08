// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package catalyst

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"math/big"
)

//go:generate go run github.com/fjl/gencodec -type assembleBlockParams -field-override assembleBlockParamsMarshaling -out gen_blockparams.go

// Structure described at https://hackmd.io/T9x2mMA4S7us8tJwEB3FDQ
type assembleBlockParams struct {
	ParentHash common.Hash `json:"parentHash"    gencodec:"required"`
	Timestamp  uint64      `json:"timestamp"     gencodec:"required"`
}

// JSON type overrides for assembleBlockParams.
type assembleBlockParamsMarshaling struct {
	Timestamp hexutil.Uint64
}

//go:generate go run github.com/fjl/gencodec -type AssembleL2BlockParams -field-override assembleL2BlockParamsMarshaling -out gen_l2blockparams.go

type AssembleL2BlockParams struct {
	Number       uint64   `json:"number"        gencodec:"required"`
	Transactions [][]byte `json:"transactions"`
}

// JSON type overrides for assembleL2BlockParams.
type assembleL2BlockParamsMarshaling struct {
	Number       hexutil.Uint64
	Transactions []hexutil.Bytes
}

//go:generate go run github.com/fjl/gencodec -type executableData -field-override executableDataMarshaling -out gen_ed.go

// Structure described at https://notes.ethereum.org/@n0ble/rayonism-the-merge-spec#Parameters1
type executableData struct {
	BlockHash    common.Hash    `json:"blockHash"     gencodec:"required"`
	ParentHash   common.Hash    `json:"parentHash"    gencodec:"required"`
	Miner        common.Address `json:"miner"         gencodec:"required"`
	StateRoot    common.Hash    `json:"stateRoot"     gencodec:"required"`
	Number       uint64         `json:"number"        gencodec:"required"`
	GasLimit     uint64         `json:"gasLimit"      gencodec:"required"`
	GasUsed      uint64         `json:"gasUsed"       gencodec:"required"`
	Timestamp    uint64         `json:"timestamp"     gencodec:"required"`
	ReceiptRoot  common.Hash    `json:"receiptsRoot"  gencodec:"required"`
	LogsBloom    []byte         `json:"logsBloom"     gencodec:"required"`
	Transactions [][]byte       `json:"transactions"  gencodec:"required"`
}

// JSON type overrides for executableData.
type executableDataMarshaling struct {
	Number       hexutil.Uint64
	GasLimit     hexutil.Uint64
	GasUsed      hexutil.Uint64
	Timestamp    hexutil.Uint64
	LogsBloom    hexutil.Bytes
	Transactions []hexutil.Bytes
}

type NewBlockResponse struct {
	Valid bool `json:"valid"`
}

type GenericResponse struct {
	Success bool `json:"success"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutableL2Data -field-override executableL2DataMarshaling -out gen_l2_ed.go

type ExecutableL2Data struct {
	// BLS message fields which need to be singed, and submitted to DA layer.
	// We chose the fields which would affect the state calculation result,
	// and the values are determined by sequencers as the BLS message.
	ParentHash   common.Hash    `json:"parentHash"     gencodec:"required"`
	Miner        common.Address `json:"miner"          gencodec:"required"`
	Number       uint64         `json:"number"         gencodec:"required"`
	GasLimit     uint64         `json:"gasLimit"       gencodec:"required"`
	BaseFee      *big.Int       `json:"baseFeePerGas"  gencodec:"required"`
	Timestamp    uint64         `json:"timestamp"      gencodec:"required"`
	Transactions [][]byte       `json:"transactions"   gencodec:"required"`

	// execution result
	StateRoot   common.Hash `json:"stateRoot"`
	GasUsed     uint64      `json:"gasUsed"`
	ReceiptRoot common.Hash `json:"receiptsRoot"`
	LogsBloom   []byte      `json:"logsBloom"`

	Extra []byte `json:"extraData"`
}

// JSON type overrides for executableL2Data.
type executableL2DataMarshaling struct {
	Number       hexutil.Uint64
	GasLimit     hexutil.Uint64
	GasUsed      hexutil.Uint64
	Timestamp    hexutil.Uint64
	LogsBloom    hexutil.Bytes
	Transactions []hexutil.Bytes
	BaseFee      *hexutil.Big
	Extra        hexutil.Bytes
}

//go:generate go run github.com/fjl/gencodec -type BLSData -field-override blsDataMarshaling -out gen_bls.go

type BLSData struct {
	BLSSigners   [][]byte `json:"bls_signers"`
	BLSSignature []byte   `json:"bls_signature"`
}

type blsDataMarshaling struct {
	BLSSigners   []hexutil.Bytes
	BLSSignature hexutil.Bytes
}
