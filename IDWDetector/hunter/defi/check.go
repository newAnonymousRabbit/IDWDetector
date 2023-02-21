package defi

import (
	"math/big"
	"encoding/json"
	"io/ioutil"
	"os"
	"fmt"
	"sort"
	"math"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

const ABNORMAL_RATE = 3
const SINGLE_ABNORMAL_RATE = 1

type TokenFlowList []TokenFlow
func (a TokenFlowList) Len() int {
	return len(a)
}

func (a TokenFlowList) Swap(i, j int){
	a[i], a[j] = a[j], a[i]
}


func (a TokenFlowList) Less(i, j int) bool {
	// from big to small in slice
	// return a[j].BlockNumber.Cmp(a[i].BlockNumber) == -1
	// from small to big
	return a[i].BlockNumber.Cmp(a[j].BlockNumber) == -1
}

type TokenTxList []TokenTx

func (a TokenTxList) Len() int {
	return len(a)
}

func (a TokenTxList) Swap(i, j int){
	a[i], a[j] = a[j], a[i]
}

// from small to big in slice
func (a TokenTxList) Less(i, j int) bool {
	if a[i].BlockNumber.Cmp(a[j].BlockNumber) == -1 {
		return true
	} else if a[i].BlockNumber.Cmp(a[j].BlockNumber) == 0{
		if a[i].TxIndex < a[j].TxIndex{
			return true
		} else if a[i].TxIndex == a[j].TxIndex{
			if a[i].EventIndex < a[j].EventIndex{
				return true
			}
		}
	}
	return false
}

type UserTokenAttackMap map[common.Address]TokenAttackMap
type TokenAttackMap map[common.Address]AttackInfo

type AttackInfo struct {
	BlockNumber *big.Int
	TotalDeposit *big.Float
	TotalWithdraw *big.Float
	TokenFlowList []TokenFlow
	RelatedUserMap map[common.Address]*big.Int
	RelatedTokenFlowMap map[common.Address][]TokenFlow
}

// 0x0000000000000000000000000000000000000000 eth
// 0x0000000000000000000000000000000000000001 stable for eth
// 0x0000000000000000000000000000000000000002 stable for USD

type UserTokenFlowMap map[common.Address]TokenFlowMap
type TokenFlowMap map[common.Address][]TokenFlow

type TokenFlow struct {
	BlockNumber *big.Int
	TxIndex int
	Amount *big.Float
	Action string
	TotalDeposit *big.Float
	TotalWithdraw *big.Float
}


// merge deposit and withdraw in a Tx
type UserTokenFlowTxMap map[common.Address]TokenFlowTxMap
type TokenFlowTxMap map[common.Address]FlowTxMap
type FlowTxMap map[string]TokenFlowTx

type TokenFlowTx struct {
	BlockNumber *big.Int
	TxIndex int
	TotalDeposit *big.Float
	TotalWithdraw *big.Float
}

func GenerateUserFlow(txExtractor *TxExtractor) (UserTokenFlowMap, map[common.Address]int) {
	unSupportToken := make(map[common.Address]int)
	userTokenFlowMap := make(UserTokenFlowMap)
	stableTokenMap := txExtractor.StableTokenMap
	LPTokenMap := txExtractor.LPTokenMap
	tokenSwapMap := txExtractor.TokenSwapMap
	zero, _ := new(big.Int).SetString("0",10)
	for userAddress, tokenTxMap := range(txExtractor.UserTokenTxMap){
		stableTokenListMap := make(map[common.Address][]TokenTx)
		tokenFlowMap := make(TokenFlowMap)
		var mergedTokenTxList []TokenTx
		for tokenAddress := range tokenTxMap{
			// collect stable token for USD and ignore the stake token
			mergedTokenTxList = append(mergedTokenTxList, tokenTxMap[tokenAddress]...)
			if _, ok := stableTokenMap[tokenAddress]; ok {
				xToken := stableTokenMap[tokenAddress].XToken
				stableTokenListMap[xToken] = append(stableTokenListMap[xToken], tokenTxMap[tokenAddress]...)
			} else {
				// ignore the stake token
				if LPTokenMap[tokenAddress] {
					continue
				}
			}
	
			// generate flow of each token
			var tokenFlowList []TokenFlow
			totalDeposit := new(big.Float)
			totalWithdraw := new(big.Float)
			tokenFlowList = append(tokenFlowList, TokenFlow{
				BlockNumber : zero,
				TxIndex : 0,
				Amount : new(big.Float),
				TotalDeposit : totalDeposit,
				TotalWithdraw : totalWithdraw,
			})
			tokenTxList := tokenTxMap[tokenAddress]
			
			sort.Sort(TokenTxList(tokenTxList))
			for _, tokenTx := range tokenTxList{
				if tokenTx.Amount.Cmp(zero) == 1{
					amount := new(big.Float).SetInt(tokenTx.Amount)
					if tokenTx.Action == "deposit"{
						totalDeposit = new(big.Float).Add(totalDeposit, amount)
					} else if tokenTx.Action == "withdraw"{
						totalWithdraw = new(big.Float).Add(totalWithdraw, amount)
					}
					tokenFlowList = append(tokenFlowList, TokenFlow{
						BlockNumber : tokenTx.BlockNumber,
						TxIndex : tokenTx.TxIndex,
						Amount : new(big.Float).Add(amount, new(big.Float)),
						Action : tokenTx.Action,
						TotalDeposit : totalDeposit,
						TotalWithdraw : totalWithdraw,
					})
				}
			}
			tokenFlowMap[tokenAddress] = tokenFlowList
		}
		// generate flow of stableToken
		for xToken := range stableTokenListMap {
			stableTokenList := stableTokenListMap[xToken]
			sort.Sort(TokenTxList(stableTokenList))
			var tokenFlowList []TokenFlow
			totalDeposit := new(big.Float)
			totalWithdraw := new(big.Float)
			tokenFlowList = append(tokenFlowList, TokenFlow{
				BlockNumber : zero,
				TxIndex : 0,
				Amount : new(big.Float),
				TotalDeposit : totalDeposit,
				TotalWithdraw : totalWithdraw,
			})
			for _, tokenTx := range stableTokenList{
				if tokenTx.Amount.Cmp(zero) == 1{
					tokenAddress := tokenTx.TokenAddress
					amount := new(big.Float).SetInt(tokenTx.Amount)
					rate := new(big.Float).SetInt64(stableTokenMap[tokenAddress].RateToXToken * int64(math.Pow(float64(10),float64(18-stableTokenMap[tokenAddress].Decimals))))
					amountForXToken := new(big.Float).Mul(amount, rate)
					if tokenTx.Action == "deposit"{
						totalDeposit = new(big.Float).Add(totalDeposit, amountForXToken)
					} else if tokenTx.Action == "withdraw"{
						totalWithdraw = new(big.Float).Add(totalWithdraw, amountForXToken)
					}
					tokenFlowList = append(tokenFlowList, TokenFlow{
						BlockNumber : tokenTx.BlockNumber,
						TxIndex : tokenTx.TxIndex,
						Amount : amountForXToken,
						Action : tokenTx.Action,
						TotalDeposit : totalDeposit,
						TotalWithdraw : totalWithdraw,
					})
				}
			}
			tokenFlowMap[xToken] = tokenFlowList
		}
	
		// deal with mergedFlow
		sort.Sort(TokenTxList(mergedTokenTxList))
		var mergedFlowList []TokenFlow
		totalDeposit := new(big.Float)
		totalWithdraw := new(big.Float)
		mergedFlowList = append(mergedFlowList, TokenFlow{
			BlockNumber : zero,
			TxIndex : 0,
			Amount : new(big.Float),
			TotalDeposit : totalDeposit,
			TotalWithdraw : totalWithdraw,
		})
		for _, tokenTx := range mergedTokenTxList{
			tokenAddress := tokenTx.TokenAddress
			if tokenTx.Amount.Cmp(zero) == 1{
				amount := new(big.Float).SetInt(tokenTx.Amount)
				var rate *big.Float
				_, inTokenSwapMap := tokenSwapMap[tokenAddress]
				_, inStableTokenMap := stableTokenMap[tokenAddress]
				if inTokenSwapMap || inStableTokenMap {
					if inStableTokenMap && stableTokenMap[tokenAddress].XToken == common.HexToAddress("0x2"){
						rate = new(big.Float).SetInt64(stableTokenMap[tokenAddress].RateToXToken * int64(math.Pow(float64(10),float64(18-stableTokenMap[tokenAddress].Decimals))))
					} else {
						if _, ok := stableTokenMap[tokenAddress]; ok {
							if stableTokenMap[tokenAddress].XToken == common.HexToAddress("0x1"){
								tokenAddress = common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
							} else if stableTokenMap[tokenAddress].XToken == common.HexToAddress("0x3"){
								tokenAddress = common.HexToAddress("0x2260fac5e5542a773aa44fbcfedf7c193bc2c599")
							}
						}
						blockNumber := tokenTx.BlockNumber
						block := blockNumber.Uint64()
						block = (block/500) * 500
						rate = new(big.Float).SetFloat64(tokenSwapMap[tokenAddress].RateMap[strconv.FormatUint(block, 10)])
					}
					amountForXToken := new(big.Float).Mul(amount, rate)
					if tokenTx.Action == "deposit"{
						totalDeposit = new(big.Float).Add(totalDeposit, amountForXToken)
					} else if tokenTx.Action == "withdraw"{
						totalWithdraw = new(big.Float).Add(totalWithdraw, amountForXToken)
					}
					mergedFlowList = append(mergedFlowList, TokenFlow{
						BlockNumber : tokenTx.BlockNumber,
						TxIndex : tokenTx.TxIndex,
						Amount : amountForXToken,
						Action : tokenTx.Action,
						TotalDeposit : totalDeposit,
						TotalWithdraw : totalWithdraw,
					})
				} else {
					unSupportToken[tokenAddress] += 1
				}
			}
		}
		tokenFlowMap[common.HexToAddress("0x4")] = mergedFlowList
		userTokenFlowMap[userAddress] = tokenFlowMap
	}
	return userTokenFlowMap, unSupportToken
}

func MergeFlowInTx(userTokenFlowMap UserTokenFlowMap) (UserTokenFlowTxMap) {
	userTokenFlowTxMap := make(UserTokenFlowTxMap)
	for userAddress := range(userTokenFlowMap){
		userTokenFlowTxMap[userAddress] = make(TokenFlowTxMap)
		for tokenAddress := range(userTokenFlowMap[userAddress]){
			userTokenFlowTxMap[userAddress][tokenAddress] = make(FlowTxMap)
			for _, tokenFlow := range(userTokenFlowMap[userAddress][tokenAddress]){
				block_tx := tokenFlow.BlockNumber.String() + "_" + strconv.Itoa(tokenFlow.TxIndex)
				if _, ok := userTokenFlowTxMap[userAddress][tokenAddress][block_tx]; !ok {
					userTokenFlowTxMap[userAddress][tokenAddress][block_tx] = TokenFlowTx{
						BlockNumber : tokenFlow.BlockNumber,
						TxIndex : tokenFlow.TxIndex,
						TotalDeposit : new(big.Float),
						TotalWithdraw : new(big.Float),
					}
				}
				tokenFlowTx := userTokenFlowTxMap[userAddress][tokenAddress][block_tx]
				if tokenFlow.Action == "deposit"{
					tokenFlowTx.TotalDeposit = tokenFlowTx.TotalDeposit.Add(tokenFlowTx.TotalDeposit, tokenFlow.Amount)
				} else if tokenFlow.Action == "withdraw"{
					tokenFlowTx.TotalWithdraw = tokenFlowTx.TotalWithdraw.Add(tokenFlowTx.TotalWithdraw, tokenFlow.Amount)
				}
			}
		}
	}
	return userTokenFlowTxMap
}

func CheckAttackInTx(userTokenFlowMap UserTokenFlowMap, txExtractor *TxExtractor) {
	zero := new(big.Float)
	userTokenFlowTxMap := MergeFlowInTx(userTokenFlowMap)
	// stableTokenMap := txExtractor.StableTokenMap
	commonAddressMap := txExtractor.CommonAddressMap
	userRelatedMap := txExtractor.UserRelatedMap
	
	for userAddress := range(userTokenFlowTxMap){
		if _, ok := commonAddressMap[userAddress]; ok {
			continue
		}
		if txExtractor.RelatedToken[userAddress] > 0{
			continue
		}
		for tokenAddress := range(userTokenFlowTxMap[userAddress]){
			// if _, ok := stableTokenMap[tokenAddress]; ok {
			// 	continue
			// }
			for block_tx, tokenFlowTx := range(userTokenFlowTxMap[userAddress][tokenAddress]){
				if tokenFlowTx.TotalDeposit.Cmp(zero) == 1{
					usedAddress := make(map[common.Address]bool)
					blockNumber := tokenFlowTx.BlockNumber
					totalDeposit := new(big.Float).Add(tokenFlowTx.TotalDeposit, new(big.Float))
					totalWithdraw := new(big.Float).Add(tokenFlowTx.TotalWithdraw, new(big.Float))
					if _, ok := userRelatedMap[userAddress]; ok {
						for relatedUser := range(userRelatedMap[userAddress]){
							if _, ok := commonAddressMap[relatedUser]; ok{
								continue
							} 
							if _, ok := usedAddress[relatedUser]; ok {
								continue
							}
							if userRelatedMap[userAddress][relatedUser].Cmp(blockNumber) <= 0 {
								usedAddress[relatedUser] = true
								relatedTotalDeposit, relatedTotalWithdraw := GetRelatedTokenFlowInTx(userTokenFlowTxMap, relatedUser, tokenAddress, userRelatedMap, usedAddress, commonAddressMap, block_tx, blockNumber)
								totalDeposit = totalDeposit.Add(totalDeposit, relatedTotalDeposit)
								totalWithdraw = totalWithdraw.Add(totalWithdraw, relatedTotalWithdraw)
							}
						}
					}
					// temp_block_interval := new(big.Int).Sub(thisTokenFlow.BlockNumber, lastDeposit.BlockNumber)
					temp_rate := new(big.Float).Quo(totalWithdraw,totalDeposit)
					rate, _ := temp_rate.Float64()
					if rate > float64(SINGLE_ABNORMAL_RATE){
						fmt.Println(userAddress, tokenAddress, totalDeposit, totalWithdraw, rate, "within one tx", tokenFlowTx.BlockNumber, tokenFlowTx.TxIndex)
					}
				}
			}
		}
	}
}

func GetRelatedTokenFlowInTx(userTokenFlowTxMap UserTokenFlowTxMap, userAddress, tokenAddress common.Address, userRelatedMap UserRelatedMap, usedAddress, commonAddressMap map[common.Address]bool, block_tx string, blockNumber *big.Int) (*big.Float, *big.Float) {
	totalDeposit := new(big.Float)
	totalWithdraw := new(big.Float)
	if _, ok := userTokenFlowTxMap[userAddress]; ok{
		if _, ok1 := userTokenFlowTxMap[userAddress][tokenAddress]; ok1{
			if _, ok2 := userTokenFlowTxMap[userAddress][tokenAddress][block_tx]; ok2{
				totalDeposit = new(big.Float).Add(userTokenFlowTxMap[userAddress][tokenAddress][block_tx].TotalDeposit, new(big.Float))
				totalWithdraw = new(big.Float).Add(userTokenFlowTxMap[userAddress][tokenAddress][block_tx].TotalWithdraw, new(big.Float))
			}
		}
	}
	if _, ok := userRelatedMap[userAddress]; ok {
		for relatedUser := range(userRelatedMap[userAddress]){
			if _, ok := commonAddressMap[relatedUser]; ok{
				continue
			} 
			if _, ok := usedAddress[relatedUser]; ok {
				continue
			}
			if userRelatedMap[userAddress][relatedUser].Cmp(blockNumber) <= 0 {
				usedAddress[relatedUser] = true
				relatedTotalDeposit, relatedTotalWithdraw := GetRelatedTokenFlowInTx(userTokenFlowTxMap, relatedUser, tokenAddress, userRelatedMap, usedAddress, commonAddressMap, block_tx, blockNumber)
				totalDeposit = totalDeposit.Add(totalDeposit, relatedTotalDeposit)
				totalWithdraw = totalWithdraw.Add(totalWithdraw, relatedTotalWithdraw)
			}
		}
	}
	return totalDeposit, totalWithdraw
}

// func abnormalDetectionInTx(totalDeposit, totalWithdraw *big.Int, actionInfo ActionInfo){
// 	flag := false
// 	zero, _ := new(big.Int).SetString("0",10)
// 	if totalDeposit.Cmp(zero) == 1{
// 		depositAmount := new(big.Float).SetInt(totalDeposit)
// 		withdrawAmount := new(big.Float).SetInt(totalWithdraw)
// 		temp_rate := new(big.Float).Quo(withdrawAmount,depositAmount)
// 		rate, _ := temp_rate.Float64()
// 		if rate >= float64(SINGLE_ABNORMAL_RATE){
// 			fmt.Println(userAddress, tokenAddress, totalDeposit, totalWithdraw, rate, "within one tx", actionInfo.BlockNumber, actionInfo.TxIndex)
// 			flag = true
// 		}
// 	}
// }

func abnormalDetection(totalDeposit, totalWithdraw *big.Float, userTokenFlowMap UserTokenFlowMap, userAddress, tokenAddress common.Address, usedAddress map[common.Address]bool, userRelatedMap UserRelatedMap, lastDeposit, thisTokenFlow TokenFlow) (AttackInfo, bool) {

	attackInfo := AttackInfo{}
	flag := false
	zero := new(big.Float)
	if totalDeposit.Cmp(zero) == 1{
		// depositAmount := new(big.Float).SetInt(totalDeposit)
		// withdrawAmount := new(big.Float).SetInt(totalWithdraw)
		// temp_block_interval := new(big.Int).Sub(thisTokenFlow.BlockNumber, lastDeposit.BlockNumber)
		temp_rate := new(big.Float).Quo(totalWithdraw,totalDeposit)
		// if thisTokenFlow.BlockNumber.Cmp(lastDeposit.BlockNumber) == 0 && thisTokenFlow.TxIndex == lastDeposit.TxIndex {
			// rate, _ := temp_rate.Float64()
			// if rate >= float64(SINGLE_ABNORMAL_RATE){
			// 	fmt.Println(userAddress, tokenAddress, totalDeposit, totalWithdraw, rate, "within one tx", thisTokenFlow.BlockNumber, thisTokenFlow.TxIndex)
			// 	flag = true
			// }
		// } else {
			// interval := new(big.Float).SetInt(temp_interval)
			// rate, _ := new(big.Float).Quo(temp_rate, interval).Float64()
			// rate = rate * float64(100000)
		rate, _ := temp_rate.Float64()
		if rate >= float64(ABNORMAL_RATE){
			fmt.Println(userAddress, tokenAddress, totalDeposit, totalWithdraw, rate)
			flag = true
		}

	} else if totalDeposit.Cmp(zero) == 0 && totalWithdraw.Cmp(zero) == 1{
		fmt.Println(userAddress, tokenAddress, totalDeposit, totalWithdraw)
		flag = true
	}
	if flag {
		attackInfo = AttackInfo{
			BlockNumber : thisTokenFlow.BlockNumber,
			TotalDeposit : totalDeposit,
			TotalWithdraw : totalWithdraw,
			TokenFlowList : userTokenFlowMap[userAddress][tokenAddress],
			RelatedUserMap : userRelatedMap[userAddress],
			RelatedTokenFlowMap : make(map[common.Address][]TokenFlow),
		}
		for relatedUser := range usedAddress {
			attackInfo.RelatedTokenFlowMap[relatedUser] = userTokenFlowMap[relatedUser][tokenAddress]
		}
	}
	return attackInfo, flag
}

func CheckAttack(userAddress common.Address, userTokenFlowMap UserTokenFlowMap, txExtractor *TxExtractor) (map[common.Address]AttackInfo, bool) {

	userRelatedMap := txExtractor.UserRelatedMap
	commonAddressMap := txExtractor.CommonAddressMap
	stableTokenMap := txExtractor.StableTokenMap
	attackInfoMap := make(map[common.Address]AttackInfo)

	flag := false
	lastDeposit := TokenFlow{
		BlockNumber : new(big.Int),
		TxIndex : 0,
	}
	if _, ok := commonAddressMap[userAddress]; ok {
		return attackInfoMap, flag
	}

	for tokenAddress := range(userTokenFlowMap[userAddress]){
		if _, ok := stableTokenMap[tokenAddress]; ok {
			continue
		}
		tokenFlowList := userTokenFlowMap[userAddress][tokenAddress]
		sort.Sort(TokenFlowList(tokenFlowList))
		for index := range(tokenFlowList) {
			tokenFlow := tokenFlowList[index]
			if tokenFlow.Action == "deposit"{
				lastDeposit = tokenFlow
				continue
			}
			totalDeposit := new(big.Float).Add(tokenFlow.TotalDeposit, new(big.Float))
			totalWithdraw := new(big.Float).Add(tokenFlow.TotalWithdraw, new(big.Float))
			blockNumber := tokenFlow.BlockNumber
			usedAddress := make(map[common.Address]bool)

			// add related users' token flow
			if _, ok := userRelatedMap[userAddress]; ok {
				for relatedUser := range(userRelatedMap[userAddress]){
					if _, ok := commonAddressMap[relatedUser]; ok{
						continue
					} 
					if _, ok := usedAddress[relatedUser]; ok {
						continue
					}
					if userRelatedMap[userAddress][relatedUser].Cmp(blockNumber) <= 0 {
						// if _, ok := userTokenFlowMap[relatedUser]; ok {
							usedAddress[relatedUser] = true
							relatedTotalDeposit, relatedTotalWithdraw := GetRelatedTokenFlow(userTokenFlowMap, relatedUser, tokenAddress, userRelatedMap, usedAddress, commonAddressMap, tokenFlow)
							totalDeposit = totalDeposit.Add(totalDeposit, relatedTotalDeposit)
							totalWithdraw = totalWithdraw.Add(totalWithdraw, relatedTotalWithdraw)
					}
				}
			}
			attackInfo, isAttack := abnormalDetection(totalDeposit, totalWithdraw, userTokenFlowMap, userAddress, tokenAddress, usedAddress, userRelatedMap, lastDeposit, tokenFlow)
			if isAttack{
				attackInfoMap[tokenAddress] = attackInfo
				flag = true
				break
			}
		}
	}
	return attackInfoMap, flag
}

func GetRelatedTokenFlow(userTokenFlowMap UserTokenFlowMap, userAddress, tokenAddress common.Address, userRelatedMap UserRelatedMap, usedAddress, commonAddressMap map[common.Address]bool, thisTokenFlow TokenFlow) (*big.Float, *big.Float) {
	totalDeposit := new(big.Float)
	totalWithdraw := new(big.Float)
	tokenFlowList := userTokenFlowMap[userAddress][tokenAddress]
	sort.Sort(TokenFlowList(tokenFlowList))
	for index := range(tokenFlowList) {
		tokenFlow := tokenFlowList[index]
		if tokenFlow.BlockNumber.Cmp(thisTokenFlow.BlockNumber) == 1 || (tokenFlow.BlockNumber.Cmp(thisTokenFlow.BlockNumber) == 0 && tokenFlow.TxIndex > thisTokenFlow.TxIndex) {
			break
		}
		// if tokenFlow.Action == "deposit"{
		// 	lastDeposit = new(big.Int).Add(tokenFlow.BlockNumber,new(big.Int))
		// }
		totalDeposit = new(big.Float).Add(tokenFlow.TotalDeposit, new(big.Float))
		totalWithdraw = new(big.Float).Add(tokenFlow.TotalWithdraw, new(big.Float))
	}
	if _, ok := userRelatedMap[userAddress]; ok {
		for relatedUser := range(userRelatedMap[userAddress]){
			if _, ok := commonAddressMap[relatedUser]; ok{
				continue
			} 
			if _, ok := usedAddress[relatedUser]; ok {
				continue
			}
			if userRelatedMap[userAddress][relatedUser].Cmp(thisTokenFlow.BlockNumber) <= 0 {
				// if _, ok := userTokenFlowMap[relatedUser]; ok {
				usedAddress[relatedUser] = true
				relatedTotalDeposit, relatedTotalWithdraw := GetRelatedTokenFlow(userTokenFlowMap, relatedUser, tokenAddress, userRelatedMap, usedAddress, commonAddressMap, thisTokenFlow)
				totalDeposit = totalDeposit.Add(totalDeposit, relatedTotalDeposit)
				totalWithdraw = totalWithdraw.Add(totalWithdraw, relatedTotalWithdraw)
			}
		}
	}
	return totalDeposit, totalWithdraw
}

type TestResult struct{
	UserTokenAttackMap UserTokenAttackMap
	ExistAttack bool
	TotalDuration string
	TestDuration string
	TestStartBlock uint64
}

func DumpTestOutput(testResult TestResult, path string){
	os.Mkdir(path, os.ModePerm)
	var err error
	b, _ := json.Marshal(testResult)
	err = ioutil.WriteFile(path + "/" + "result.json", b, 0644)
	if err != nil{
		fmt.Println(err)
	}
}


