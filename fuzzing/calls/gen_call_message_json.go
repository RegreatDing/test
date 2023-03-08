// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package calls

import (
	"encoding/json"
	"github.com/pkg/errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var _ = (*callMessageMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (c CallMessage) MarshalJSON() ([]byte, error) {
	type CallMessage struct {
		MsgFrom          common.Address            `json:"from"`
		MsgTo            *common.Address           `json:"to"`
		MsgNonce         uint64                    `json:"nonce"`
		MsgValue         *hexutil.Big              `json:"value"`
		MsgGas           uint64                    `json:"gas"`
		MsgGasPrice      *hexutil.Big              `json:"gasPrice"`
		MsgGasFeeCap     *hexutil.Big              `json:"gasFeeCap"`
		MsgGasTipCap     *hexutil.Big              `json:"gasTipCap"`
		MsgData          hexutil.Bytes             `json:"data,omitempty"`
		MsgDataAbiValues *CallMessageDataAbiValues `json:"dataAbiValues,omitempty"`
	}
	var enc CallMessage
	enc.MsgFrom = c.MsgFrom
	enc.MsgTo = c.MsgTo
	enc.MsgNonce = c.MsgNonce
	enc.MsgValue = (*hexutil.Big)(c.MsgValue)
	enc.MsgGas = c.MsgGas
	enc.MsgGasPrice = (*hexutil.Big)(c.MsgGasPrice)
	enc.MsgGasFeeCap = (*hexutil.Big)(c.MsgGasFeeCap)
	enc.MsgGasTipCap = (*hexutil.Big)(c.MsgGasTipCap)
	enc.MsgData = c.MsgData
	enc.MsgDataAbiValues = c.MsgDataAbiValues

	b, err := json.Marshal(&enc)
	return b, errors.WithStack(err)
}

// UnmarshalJSON unmarshals from JSON.
func (c *CallMessage) UnmarshalJSON(input []byte) error {
	type CallMessage struct {
		MsgFrom          *common.Address           `json:"from"`
		MsgTo            *common.Address           `json:"to"`
		MsgNonce         *uint64                   `json:"nonce"`
		MsgValue         *hexutil.Big              `json:"value"`
		MsgGas           *uint64                   `json:"gas"`
		MsgGasPrice      *hexutil.Big              `json:"gasPrice"`
		MsgGasFeeCap     *hexutil.Big              `json:"gasFeeCap"`
		MsgGasTipCap     *hexutil.Big              `json:"gasTipCap"`
		MsgData          *hexutil.Bytes            `json:"data,omitempty"`
		MsgDataAbiValues *CallMessageDataAbiValues `json:"dataAbiValues,omitempty"`
	}
	var dec CallMessage
	if err := json.Unmarshal(input, &dec); err != nil {
		return errors.WithStack(err)
	}
	if dec.MsgFrom != nil {
		c.MsgFrom = *dec.MsgFrom
	}
	if dec.MsgTo != nil {
		c.MsgTo = dec.MsgTo
	}
	if dec.MsgNonce != nil {
		c.MsgNonce = *dec.MsgNonce
	}
	if dec.MsgValue != nil {
		c.MsgValue = (*big.Int)(dec.MsgValue)
	}
	if dec.MsgGas != nil {
		c.MsgGas = *dec.MsgGas
	}
	if dec.MsgGasPrice != nil {
		c.MsgGasPrice = (*big.Int)(dec.MsgGasPrice)
	}
	if dec.MsgGasFeeCap != nil {
		c.MsgGasFeeCap = (*big.Int)(dec.MsgGasFeeCap)
	}
	if dec.MsgGasTipCap != nil {
		c.MsgGasTipCap = (*big.Int)(dec.MsgGasTipCap)
	}
	if dec.MsgData != nil {
		c.MsgData = *dec.MsgData
	}
	if dec.MsgDataAbiValues != nil {
		c.MsgDataAbiValues = dec.MsgDataAbiValues
	}
	return nil
}
