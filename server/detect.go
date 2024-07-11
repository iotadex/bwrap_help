package server

import (
	"bhelp/config"
	"bhelp/evm"
	"bhelp/gl"
	"bhelp/model"
	"encoding/hex"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var ethClient *evm.EvmToken
var btcClient *evm.EvmToken

func ListenTokens() {
	contracts := make([]common.Address, 2)
	contracts[0] = common.HexToAddress("0x7C32097EB6bA75Dc5eF370BEC9019FD09D96ab9d") //ETH
	contracts[1] = common.HexToAddress("0x6c2F73072bD9bc9052D99983e36411f48fa6cDf0") //BTC

	rpc := config.Tokens["ETH"].NodeRpc
	wss := config.Tokens["ETH"].NodeWss
	var err error
	ethClient, err = evm.NewEvmToken(rpc, wss, "0x7C32097EB6bA75Dc5eF370BEC9019FD09D96ab9d", config.Tokens["ETH"].Account)
	if err != nil {
		panic(err)
	}
	btcClient, err = evm.NewEvmToken(rpc, wss, "0x6c2F73072bD9bc9052D99983e36411f48fa6cDf0", config.Tokens["ETH"].Account)
	if err != nil {
		panic(err)
	}

	listenUnWrap("0xf98eCe9c7d0f241dA91b9895fbe4ebbc591D1CBe", config.Tokens["sETH"].Account)
}

func listenUnWrap(addr string, account common.Address) {
	smrRpcUrl := "https://json-rpc.evm.shimmer.network"
	smrWssUrl := "wss://ws.json-rpc.evm.shimmer.network"
	con, err := evm.NewEvmToken(smrRpcUrl, smrWssUrl, addr, account)
	if err != nil {
		panic(err)
	}
	for {
		orderC := make(chan *evm.UnwrapOrder, 10)
		go con.StartListen(orderC)
		gl.OutLogger.Info("Begin to listen bridge help. %s", account.Hex())
		for order := range orderC {
			if order.Error != nil {
				gl.OutLogger.Error(order.Error.Error())
				if order.Type == 0 {
					break
				}
			} else {
				gl.OutLogger.Info("UnWrap Order : %v", *order)
				if order.Org == "native" {
					dealWithdrawOrder(con, order)
				} else {
					dealUnWrapOrder(order)
				}
			}
		}
		time.Sleep(time.Second * 5)
		gl.OutLogger.Error("try to connect node again.")
	}
}

func dealUnWrapOrder(order *evm.UnwrapOrder) {
	wo := model.SwapOrder{
		TxID:      order.TxID,
		SrcToken:  order.ToToken,
		DestToken: "unKnown",
		Wrap:      -1,
		From:      order.From,
		To:        order.To,
		Amount:    order.Amount.String(),
		Ts:        time.Now().UnixMilli(),
		Org:       order.Org,
	}

	// Check the chain tx
	if err := model.StoreSwapOrder(&wo); err != nil {
		if !strings.HasPrefix(err.Error(), "Error 1062") {
			gl.OutLogger.Error("store the unwrap order to db error(%v). %v", err, wo)
		}
	}

	// Get Private Key
	_, prv, err := config.GetPrivateKey("ETH")
	if err != nil {
		gl.OutLogger.Error("GetPrivateKey error. ETH, %v", err)
		return
	}

	t := ethClient
	if order.ToToken != "ETH" {
		t = btcClient
	}

	id, err := t.SendUnWrap(order.TxID, order.Amount, order.To, prv)
	if err != nil {
		gl.OutLogger.Error("SendUnWrap error. %s, %v", order.TxID, err)
		return
	}
	gl.OutLogger.Info("SendUnWrap. unKnown => %s OK. %s", order.ToToken, hex.EncodeToString(id))
}

func dealWithdrawOrder(t1 *evm.EvmToken, order *evm.UnwrapOrder) {
	// Get Private Key
	_, prv, err := config.GetPrivateKey("ETH")
	if err != nil {
		gl.OutLogger.Error("GetPrivateKey error. ETH, %v", err)
		return
	}

	t := t1
	if order.ToToken == "ETH" {
		t = ethClient
	}

	id, err := t.SendEth(order.Amount, order.To, prv)
	if err != nil {
		gl.OutLogger.Error("Withdraw native error. %s, %v", order.TxID, err)
		return
	}
	gl.OutLogger.Info("Withdraw native %s OK. %s", order.ToToken, hex.EncodeToString(id))
}
