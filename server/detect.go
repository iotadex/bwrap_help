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

func ListenTokens() {
	contracts := make([]common.Address, 5)
	contracts[0] = common.HexToAddress("0x7C32097EB6bA75Dc5eF370BEC9019FD09D96ab9d") //ETH
	contracts[1] = common.HexToAddress("0xa158A39d00C79019A01A6E86c56E96C461334Eb0") //sETH
	contracts[2] = common.HexToAddress("0x6c2F73072bD9bc9052D99983e36411f48fa6cDf0") //BTC
	contracts[3] = common.HexToAddress("0x1cDF3F46DbF8Cf099D218cF96A769cea82F75316") //sBTC
	contracts[4] = common.HexToAddress("0x5dA63f4456A56a0c5Cb0B2104a3610D5CA3d48E8") //sIOTA

	//help := common.HexToAddress("")
	listenUnWrap("0x093B53bA1DF48D1D037e2528a682A14D88Be5c3C", config.Tokens["sETH"].Account)
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
				dealUnWrapOrder(con, order)
			}
		}
		time.Sleep(time.Second * 5)
		gl.OutLogger.Error("try to connect node again.")
	}
}

func dealUnWrapOrder(t1 *evm.EvmToken, order *evm.UnwrapOrder) {
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

	id, err := t1.SendUnWrap(order.TxID, order.Amount, order.To, prv)
	if err != nil {
		gl.OutLogger.Error("SendUnWrap error. %s, %v", order.TxID, err)
		return
	}
	gl.OutLogger.Info("SendUnWrap. unKnown => %s OK. %s", order.ToToken, hex.EncodeToString(id))
}
