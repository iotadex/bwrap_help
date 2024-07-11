package evm

import (
	"bhelp/gl"
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var EventUnWrap = crypto.Keccak256Hash([]byte("UnWrap(address,address,bytes32,uint256)"))
var MethodSend = crypto.Keccak256Hash([]byte("send(bytes32,uint256,address)"))
var zeroAddress common.Address

type UnwrapOrder struct {
	TxID    string
	ToToken string
	From    string
	To      string
	Amount  *big.Int
	Org     string // tag the platform, "IotaBee", "TangleSwap"
	Error   error
	Type    int // 0 need to reconnect and 1 only need to record
}

type HelpOrder struct {
	TxID      string
	Signer    string
	Direction int8
	Count     uint64
	Number    *big.Int
	Error     error
	Type      int // 0 need to reconnect and 1 only need to record
}

type EvmToken struct {
	client   *ethclient.Client
	rpc      string
	wss      string
	chainId  *big.Int
	contract common.Address
	account  common.Address
}

func NewEvmToken(rpc, wss, conAddr string, _account common.Address) (*EvmToken, error) {
	c, err := ethclient.Dial(rpc)
	if err != nil {
		return nil, err
	}
	chainId, err := c.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}

	return &EvmToken{
		rpc:      rpc,
		wss:      wss,
		client:   c,
		chainId:  chainId,
		contract: common.HexToAddress(conAddr),
		account:  _account,
	}, err
}

func (ei *EvmToken) SendUnWrap(txid string, amount *big.Int, to string, prv *ecdsa.PrivateKey) ([]byte, error) {
	toAddr := common.HexToAddress(to)
	if bytes.Equal(toAddr[:], zeroAddress[:]) {
		return nil, fmt.Errorf("to address error. %s", to)
	}

	txHash := common.FromHex(txid)
	if len(txHash) > 32 {
		txHash = txHash[:32]
	}
	var data []byte
	data = append(data, MethodSend[:4]...)
	data = append(data, common.LeftPadBytes(txHash, 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.FromHex(to), 32)...)
	value := big.NewInt(0)

	nonce, err := ei.client.PendingNonceAt(context.Background(), ei.account)
	if err != nil {
		return nil, err
	}

	gasPrice, err := ei.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get SuggestGasPrice error. %v", err)
	}

	tx := types.NewTransaction(nonce, ei.contract, value, gl.GasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(ei.chainId), prv)
	if err != nil {
		return nil, err
	}

	err = ei.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, err
	}

	return signedTx.Hash().Bytes(), nil
}

func (ei *EvmToken) StartListen(ch chan *UnwrapOrder) {
	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{ei.contract},
	}

	errOrder := &UnwrapOrder{Type: 0}

	//Create the ethclient
	c, err := ethclient.Dial(ei.wss)
	if err != nil {
		errOrder.Error = fmt.Errorf("the EthWssClient redial error. %v\nThe EthWssClient will be redialed later", err)
		ch <- errOrder
		return
	}
	eventLogChan := make(chan types.Log)
	sub, err := c.SubscribeFilterLogs(context.Background(), query, eventLogChan)
	if err != nil || sub == nil {
		errOrder.Error = fmt.Errorf("get event logs from eth wss client error. %v", err)
		ch <- errOrder
		return
	}
	for {
		select {
		case err := <-sub.Err():
			errOrder.Error = fmt.Errorf("event wss sub error. %v\nThe EthWssClient will be redialed later", err)
			ch <- errOrder
			return
		case vLog := <-eventLogChan:
			if vLog.Topics[0].Hex() == EventUnWrap.Hex() {
				ei.dealUnWrapEvent(ch, &vLog)
			}
		}
	}
}

func (ei *EvmToken) dealUnWrapEvent(ch chan *UnwrapOrder, vLog *types.Log) {
	errOrder := &UnwrapOrder{Type: 1}
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) == 0 {
		errOrder.Error = fmt.Errorf("unWrap event data is nil. %s, %s, %s", tx, vLog.Address.Hex(), vLog.Topics[1].Hex())
		ch <- errOrder
		return
	}
	symbol, _, _ := bytes.Cut(vLog.Data[:32], []byte{0})
	amount := new(big.Int).SetBytes(vLog.Data[32:])

	order := &UnwrapOrder{
		TxID:    tx,
		ToToken: string(symbol),
		From:    common.BytesToAddress(vLog.Topics[1][:]).Hex(),
		To:      common.BytesToAddress(vLog.Topics[2][:]).Hex(),
		Amount:  amount,
		Org:     "IotaBee",
	}
	ch <- order
}

/*
var EventSubSigner = crypto.Keccak256Hash([]byte("SubSigner(address,address,uint256)"))
var EventAddSigner = crypto.Keccak256Hash([]byte("AddSigner(address,address,uint256)"))
var EventChangeCount = crypto.Keccak256Hash([]byte("ChangeCount(address,uint8,uint256)"))
func (ei *EvmToken) dealSubSignerEvent(ch chan *HelpOrder, vLog *types.Log) {
	errOrder := &HelpOrder{Type: 1}
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) == 0 {
		errOrder.Error = fmt.Errorf("unWrap event data is nil. %s, %s, %s", tx, vLog.Address.Hex(), vLog.Topics[1].Hex())
		ch <- errOrder
		return
	}

	order := &HelpOrder{
		TxID:      tx,
		Signer:    common.BytesToAddress(vLog.Topics[2][:]).Hex(),
		Direction: -1,
		Count:     0,
		Number:    new(big.Int).SetBytes(vLog.Data[:32]),
	}
	ch <- order
}

func (ei *EvmToken) dealAddSignerEvent(ch chan *HelpOrder, vLog *types.Log) {
	errOrder := &HelpOrder{Type: 1}
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) == 0 {
		errOrder.Error = fmt.Errorf("unWrap event data is nil. %s, %s, %s", tx, vLog.Address.Hex(), vLog.Topics[1].Hex())
		ch <- errOrder
		return
	}

	order := &HelpOrder{
		TxID:      tx,
		Signer:    common.BytesToAddress(vLog.Topics[2][:]).Hex(),
		Direction: -1,
		Count:     0,
		Number:    new(big.Int).SetBytes(vLog.Data[:32]),
	}
	ch <- order
}
*/
