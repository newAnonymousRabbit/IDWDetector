package main

import (
	"fmt"
	defi "github.com/ethereum/go-ethereum/hunter/defi"
	"github.com/ethereum/go-ethereum/common"
)

func main(){
	dapp := "bzx"
	fmt.Println("Start", dapp)
	block := uint64(13000000)
	
	proxyInfoMap := defi.LoadDefi("hunter/defi.csv", false, false)
	txExtractor := defi.TxExtractor{
		Dapp: dapp,
		ActionMap: defi.LoadActionMap("hunter/defi/ActionMap.json"),
		ProxyInfo : proxyInfoMap[dapp],
		UserTokenTxMap : make(defi.UserTokenTxMap),
		UserRelatedMap : make(defi.UserRelatedMap),
		AddressCountMap : make(map[common.Address]int),
		ActionInfoList : make(map[int]defi.ActionInfo),
		MethodCountMap : make(map[string]uint),
		RelatedToken : make(map[common.Address]int),
		StableTokenMap : defi.LoadStableTokenInfo("hunter/stableToken"),
		TokenSwapMap : defi.LoadTokenSwapInfo("hunter/token.json", "hunter/priceData/usdt"),
	}
	txExtractor.Init()

	defi.InitTokenUserTxMap(block, txExtractor, "hunter/defi_apps/")
	defi.InitUserRelatedMapWithStake(block, txExtractor, "hunter/defi_apps/")
	fmt.Println("len of txExtractor.ActionInfoList", len(txExtractor.ActionInfoList))
	txExtractor.UpdateCommonAddressMap()
	txExtractor.ExtractActionInfoList()

	fmt.Println("userTokenTxMap", len(txExtractor.UserTokenTxMap))

	unSupportToken := make(map[common.Address]int)
	userTokenFlowMap, unSupportToken := defi.GenerateUserFlow(&txExtractor)

	// check attack in single tx
	fmt.Println("Check attack in single tx, mode", txExtractor.FlowMergeMode)
	defi.CheckAttackInTx(userTokenFlowMap, &txExtractor)

	
	fmt.Println("Check attack cross tx, mode", txExtractor.FlowMergeMode)
	for userAddress := range(userTokenFlowMap){
		if !txExtractor.CommonAddressMap[userAddress] && txExtractor.RelatedToken[userAddress] == 0 {
			defi.CheckAttack(userAddress, userTokenFlowMap, &txExtractor)
		}
	}
}


