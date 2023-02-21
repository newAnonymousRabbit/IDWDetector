package defi

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	vm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type TxCollector struct {
	ProxyMap map[string]string
	StakeMap map[string]string
}

// find related txs and dump json files
func (txCollector *TxCollector) ParseTxAndDump(ExTx *vm.ExternalTx, dumpBool bool){
	proxy_dump_dict := make(map[string]string)
	stake_dump_dict := make(map[string]string)

	if printCall{
		fmt.Println("Sender", ExTx.InTxs[0].From)
	}
	if len(ExTx.InTxs) == 1{
		txCollector.ParseTxTreeUtil(ExTx.InTxs[0], 0, proxy_dump_dict, stake_dump_dict)
	}

	if dumpBool {
		for proxy := range(proxy_dump_dict){
			ExTx.DumpTree("hunter/dataset/" + proxy_dump_dict[proxy] + "/proxy")
		}
		for stake := range(stake_dump_dict){
			fmt.Println(ExTx, stake, stake_dump_dict[stake])
			ExTx.DumpTree("hunter/dataset/" + stake_dump_dict[stake] + "/stake")
		}
	}
}

func (txCollector *TxCollector) ParseTxTreeUtil(InTx *vm.InternalTx, depth int, proxy_dump_dict,stake_dump_dict map[string]string){
	// printCall
	if printCall{
		var functionSiginature string
		if len(hex.EncodeToString(InTx.Input)) >= 8{
			functionSiginature = "0x" + hex.EncodeToString(InTx.Input)[:8]
		}
		if InTx.Value != nil {
			fmt.Println(strings.Repeat("-",depth+1),depth, InTx.CallType, InTx.To, functionSiginature, InTx.Value)
		} else {
			fmt.Println(strings.Repeat("-",depth+1),depth, InTx.CallType, InTx.To, functionSiginature)
		}
	}
	callTo := strings.ToLower(InTx.To.String())
	// if the callTo is related to proxy or stake token, add it 
	if _, ok := txCollector.ProxyMap[callTo]; ok {
		proxy_dump_dict[callTo] = txCollector.ProxyMap[callTo]
	}

	if _, ok := txCollector.StakeMap[callTo]; ok && InTx.CallType != "StaticCall" {
		stake_dump_dict[callTo] = txCollector.StakeMap[callTo]
	}
	
	// the event before
	for _, Tx := range InTx.InTxs {
		txCollector.ParseTxTreeUtil(Tx, depth+1, proxy_dump_dict, stake_dump_dict)
	}

	for _, event := range(InTx.Events) {
		txCollector.parseEvent(event, depth, proxy_dump_dict, stake_dump_dict)
	}
}

func (txCollector *TxCollector) parseEvent(event *vm.Event, depth int, proxy_dump_dict, stake_dump_dict map[string]string) {

	// fmt.Println(strings.Repeat("-",depth+2), depth+1, "event", event.Index)
	// identify erc20 transfer
	if len(event.Topics) > 0 && event.Topics[0].String() == "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" && len(event.Topics) == 3{		
		parsed, err := abi.JSON(strings.NewReader(erc20string))
		if err != nil {
			fmt.Println("Parser error", err)
		}
		res, err := parsed.Unpack("Transfer", event.Data)
		if err != nil {
			fmt.Println("Parser error", err)
		}
		// fmt.Println(res,err)
		amount := res[0]
		sender := strings.ToLower(common.HexToAddress(event.Topics[1].String()).String())
		to := strings.ToLower(common.HexToAddress(event.Topics[2].String()).String())
		if printEvent {
			fmt.Println(strings.Repeat("-",depth+2), depth+1, "event", event.Address, "Transfer from", sender, "to", to, "amount", amount)
		}
		// if the sender or to is related to proxy, add it
		if _, ok := txCollector.ProxyMap[sender]; ok {
			proxy_dump_dict[sender] = txCollector.ProxyMap[sender]
		} else if _, ok := txCollector.ProxyMap[to]; ok {
			proxy_dump_dict[to] = txCollector.ProxyMap[to]
		}
	}
}