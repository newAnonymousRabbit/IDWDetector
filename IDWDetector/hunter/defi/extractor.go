package defi

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"math/big"

	common "github.com/ethereum/go-ethereum/common"
	vm "github.com/ethereum/go-ethereum/core/vm"
	abi "github.com/ethereum/go-ethereum/accounts/abi"
)

type UserTokenTxMap map[common.Address]TokenTxMap // userAddress => TokenTxMap
type TokenTxMap map[common.Address][]TokenTx // tokenAddress => []TokenTx

type TokenTx struct {
	BlockNumber *big.Int
	TxIndex int
	EventIndex uint
	Sender common.Address
	To common.Address
	Amount *big.Int
	TokenAddress common.Address
	
	Action string
}

type UserRelatedMap map[common.Address]RelatedMap
type RelatedMap map[common.Address]*big.Int

func LoadABIJSON(path string) abi.ABI {
	jsonFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}	
	defer jsonFile.Close()
	abiByte, _ := ioutil.ReadAll(jsonFile)
	temp_abi, _ := abi.JSON(strings.NewReader(string(abiByte)))
	return temp_abi
}

func LoadActionMap(path string) map[string]string {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("Load action json error:", err)
	}
	temp_map := make(map[string][]string)
	err = json.Unmarshal([]byte(file), &temp_map)
	if err != nil {
		fmt.Println("Json to struct error:",err)
	}
	actionMap := make(map[string]string)
	for action := range temp_map{
		for _, name := range temp_map[action]{
			actionMap[name] = action
		}
	}
	// fmt.Println(actionMap)
	return actionMap
}

type TxExtractor struct {
	Dapp string
	ActionMap map[string]string
	ProxyInfo ProxyInfo
	ProxyABI map[common.Address] abi.ABI
	UserTokenTxMap UserTokenTxMap
	UserRelatedMap UserRelatedMap

	ActionInfoList map[int]ActionInfo
	AddressCountMap map[common.Address]int
	CommonAddressMap map[common.Address]bool
	RelatedToken map[common.Address]int
	FlowMergeMode string // single: single token; usd: pledged to usd; eth: pledged to eth; btc: pledged to btc; mix: conver to usd;
	
	MethodCountMap map[string]uint
	LPTokenMap map[common.Address]bool
	StableTokenMap map[common.Address]StableToken
	TokenSwapMap map[common.Address]TokenSwap
}

type ActionInfo struct {
	Action string
	BlockNumber *big.Int
	TxIndex int
	TokenTxList []TokenTx
}

func (txExtractor *TxExtractor) Init(){
	txExtractor.ProxyABI = make(map[common.Address] abi.ABI)
	for _, proxy := range txExtractor.ProxyInfo.ProxyContracts{
		txExtractor.ProxyABI[proxy] = LoadABIJSON("hunter/defi_apps/" + txExtractor.Dapp + "/" + strings.ToLower(proxy.String()) + ".json")
	}
	txExtractor.LPTokenMap = make(map[common.Address] bool)
	for _, token := range txExtractor.ProxyInfo.LPTokens{
		txExtractor.LPTokenMap[token] = true
	}
}

func (txExtractor *TxExtractor) extractAction(input []byte, proxy common.Address) (action string) {
	abi := txExtractor.ProxyABI[proxy]
	method, err := abi.MethodById(input)

	if err != nil{
		// fmt.Println(err)
	} else {
		methodString := method.Name + "(0x" + hex.EncodeToString(method.ID) + ")"
		txExtractor.MethodCountMap[methodString] += 1
	}
	action = "none"
	if err == nil {
		for name := range txExtractor.ActionMap {
			if strings.Contains(strings.ToLower(method.Name), name) {
				temp_action := txExtractor.ActionMap[name]
				if temp_action == "none"{
					action = "none"
					break
				} else {
					action = temp_action
				}
			}
		}
	}
	return action
}

// analyze a tx and extract token tranfer information
func (txExtractor *TxExtractor) ExtractTokenTx(ExTx *vm.ExternalTx) {
	// fmt.Println(ExTx.BlockNumber, ExTx.Timestamp)
	// SetCallVisibility(true)
	// SetEventVisibility(true)
	// ParseTxAndDump(ExTx, false)
	if len(ExTx.InTxs) == 1{
		txExtractor.extractInTxInfo(ExTx, ExTx.InTxs[0])
	}
}

func (txExtractor *TxExtractor) extractInTxInfo(ExTx *vm.ExternalTx, InTx *vm.InternalTx) (tokenTxList []TokenTx) {

	action := "none"
	if _, ok := txExtractor.ProxyABI[InTx.To]; ok && len(InTx.Input) >= 4{
		action = txExtractor.extractAction(InTx.Input, InTx.To)
		// if ExTx.BlockNumber.Cmp(big.NewInt(int64(11447264))) == 0{
		// 	fmt.Println(InTx.From, InTx.To, InTx.Value, hex.EncodeToString(InTx.Input[:4]))
		// }
	}
	// flag = true

	// extract ether transfer
	// delegatecall and static call have no Value
	if InTx.Value != nil && InTx.Value.Cmp(new(big.Int)) > 0 {
		// if ExTx.BlockNumber.Cmp(big.NewInt(int64(11447264))) == 0{
		// 	fmt.Println(InTx.From, InTx.To, InTx.Value, action)
		// }
		// fmt.Println(InTx.From, InTx.To, InTx.Value)
		tokenTxList = append(tokenTxList, TokenTx{
			Sender : InTx.From,
			To : InTx.To,
			Amount : new(big.Int).Add(InTx.Value, new(big.Int)),
			TokenAddress : common.HexToAddress("0x0"),
		})
	}
	for _, Tx := range InTx.InTxs {
		temp_tokenTxList := txExtractor.extractInTxInfo(ExTx, Tx)
		tokenTxList = append(tokenTxList, temp_tokenTxList...)
	}

	// check the function is deposit or withdraw
	for _, event := range(InTx.Events) {
		tokenTx, isTokenTx := extractEvent(event)
		if isTokenTx{
			tokenTxList = append(tokenTxList, tokenTx)
		}
	}
	if action != "none"{
		// fmt.Println(functionSiginature)
		zero, _ := new(big.Int).SetString("0",10)
		var zeroAddress common.Address
		for _, tokenTx := range(tokenTxList){
			if _, ok := txExtractor.StableTokenMap[tokenTx.TokenAddress]; !ok && (tokenTx.Sender == zeroAddress || tokenTx.To == zeroAddress) {
				txExtractor.LPTokenMap[tokenTx.TokenAddress] = true
			} else {
				txExtractor.AddressCountMap[tokenTx.Sender] += 1
				txExtractor.AddressCountMap[tokenTx.To] += 1
			}
		}
		actionInfo := ActionInfo{
			Action : action,
			BlockNumber : new(big.Int).Add(ExTx.BlockNumber, zero),
			TxIndex : ExTx.TxIndex,
			TokenTxList : tokenTxList,
		}
		txExtractor.ActionInfoList[len(txExtractor.ActionInfoList)] = actionInfo
		var emptyTokenTxList []TokenTx
		return emptyTokenTxList
	}
	return tokenTxList
}

func (txExtractor *TxExtractor) UpdateCommonAddressMap(){
	length := len(txExtractor.ActionInfoList)
	txExtractor.CommonAddressMap = make(map[common.Address]bool)
	for proxy := range txExtractor.ProxyABI{
		txExtractor.CommonAddressMap[proxy] = true
	}
	for address := range txExtractor.AddressCountMap{
		if txExtractor.AddressCountMap[address] > 350 || float64(txExtractor.AddressCountMap[address]) > float64(length) * float64(0.8) {
			fmt.Println(address, txExtractor.AddressCountMap[address], len(txExtractor.ActionInfoList))
			txExtractor.CommonAddressMap[address] = true
		}
	}
	// fmt.Println("commonAddressMap", txExtractor.CommonAddressMap)
}

func (txExtractor *TxExtractor) ExtractActionInfoList(){
	var zeroAddress common.Address
	userTokenTxMap := txExtractor.UserTokenTxMap
	userRelatedMap := txExtractor.UserRelatedMap
	for _, actionInfo := range txExtractor.ActionInfoList{

		var depositTo []common.Address
		var withdrawFrom []common.Address
		var TokenTxListWithoutLP []TokenTx
		for _, tokenTx := range(actionInfo.TokenTxList){
			sender := tokenTx.Sender
			to := tokenTx.To
			tokenAddress := tokenTx.TokenAddress
			// isMintOrBurn := false
			if _, ok := txExtractor.LPTokenMap[tokenAddress]; ok {
				if sender == zeroAddress{
					// deposit
					// isMintOrBurn = true
					depositTo = append(depositTo, to)
				} else if to == zeroAddress{
					// withdraw
					// isMintOrBurn = true
					withdrawFrom = append(withdrawFrom, sender)
				}
			} else {
			// if isMintOrBurn == false {
				TokenTxListWithoutLP = append(TokenTxListWithoutLP, tokenTx)
			}
		}

		for _, tokenTx := range(TokenTxListWithoutLP){
			tokenTx.TxIndex = actionInfo.TxIndex
			tokenTx.BlockNumber = actionInfo.BlockNumber
			tokenTx.Action = actionInfo.Action

			tokenAddress := tokenTx.TokenAddress
			sender := tokenTx.Sender
			to := tokenTx.To
			if actionInfo.Action == "withdraw" {
				if txExtractor.CommonAddressMap[sender] {
					if _, ok := userTokenTxMap[to]; !ok {
						userTokenTxMap[to] = make(TokenTxMap)
					}
					userTokenTxMap[to][tokenAddress] = append(userTokenTxMap[to][tokenAddress], tokenTx)
					// handle the case that A withdraw from B
					for _, relatedUser := range(withdrawFrom){
						updateUserRelatedMap(userRelatedMap, to, relatedUser, actionInfo.BlockNumber)
					}
					txExtractor.RelatedToken[tokenAddress] += 1
				}
			} else if actionInfo.Action == "deposit" {
				if txExtractor.CommonAddressMap[to] {
					if _, ok := userTokenTxMap[sender]; !ok {
						userTokenTxMap[sender] = make(TokenTxMap)
					}
					userTokenTxMap[sender][tokenAddress] = append(userTokenTxMap[sender][tokenAddress], tokenTx)

					// handle the case that A deposit to B
					for _, relatedUser := range(depositTo){
						updateUserRelatedMap(userRelatedMap, relatedUser, sender, actionInfo.BlockNumber)
					}
					txExtractor.RelatedToken[tokenAddress] += 1
				}
			}
		}
	}


	mode := "single"
	modeMap := make(map[string]int)
	for token := range txExtractor.RelatedToken{
		if txExtractor.StableTokenMap[token].XToken == common.HexToAddress("0x1"){
			modeMap["eth"] += 1
		} else if txExtractor.StableTokenMap[token].XToken == common.HexToAddress("0x2"){
			modeMap["usd"] += 1
		} else if txExtractor.StableTokenMap[token].XToken == common.HexToAddress("0x3"){
			modeMap["btc"] += 1
		} else {
			modeMap["others"] += 1
		}
	}
	if len(modeMap) == 1{
		if modeMap["eth"] > 0{
			mode = "eth"
		} else if modeMap["usd"] > 0{
			mode = "usd"
		} else if modeMap["btc"] > 0{
			mode = "btc"
		} else if modeMap["others"] > 1{
			mode = "mix"
		}
	} else {
		mode = "mix"
	}
	txExtractor.FlowMergeMode = mode
	fmt.Println("tokens", len(txExtractor.RelatedToken), mode)
}

func updateUserRelatedMap(userRelatedMap UserRelatedMap, mainUser common.Address, relatedUser common.Address, blockNumber *big.Int){
	if mainUser != relatedUser{
		if _, ok := userRelatedMap[mainUser]; !ok{
			userRelatedMap[mainUser] = make(RelatedMap)
		}
		if _, ok := (userRelatedMap[mainUser][relatedUser]); !ok{
			userRelatedMap[mainUser][relatedUser] = blockNumber
		} else if userRelatedMap[mainUser][relatedUser].Cmp(blockNumber) == 1 {
			userRelatedMap[mainUser][relatedUser] = blockNumber
		}
	}
}

func (txExtractor *TxExtractor) ExtractStakeTokenTx(ExTx *vm.ExternalTx){
	if len(ExTx.InTxs) == 1{
		txExtractor.extractInTxForStake(ExTx, ExTx.InTxs[0])
	}
}

func (txExtractor *TxExtractor) extractInTxForStake(ExTx *vm.ExternalTx, InTx *vm.InternalTx) {
	userRelatedMap := txExtractor.UserRelatedMap
	for _, Tx := range InTx.InTxs {
		txExtractor.extractInTxForStake(ExTx, Tx)
	}

	for _, event := range(InTx.Events) {
		tokenTx, isTokenTx := extractEvent(event)
		if isTokenTx{
			if _, ok := txExtractor.LPTokenMap[tokenTx.TokenAddress]; ok {
				sender := tokenTx.Sender
				to := tokenTx.To
				var zeroAddress common.Address
				if sender != zeroAddress && to != zeroAddress{
					// add sender in to's related users
					updateUserRelatedMap(userRelatedMap, to, sender, ExTx.BlockNumber)
				}
			}
		}
	}
}

func extractEvent(event *vm.Event) (TokenTx, bool) {
	
	var tokenTx TokenTx
	var isTokenTx bool
	if len(event.Topics) > 0 && event.Topics[0].String() == "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" && len(event.Topics) == 3{
		// parsed, err := abi.JSON(strings.NewReader(ERC20ABI))
		parsed, err := abi.JSON(strings.NewReader(erc20string))
		if err != nil {
			fmt.Println("Parser error", err)
		}
		res, err := parsed.Unpack("Transfer", event.Data)
		if err != nil {
			fmt.Println("Parser error", err)
		}
		// fmt.Println(reflect.TypeOf(res[0]),err)
		sender := common.HexToAddress(event.Topics[1].String())
		to := common.HexToAddress(event.Topics[2].String())
		amount, _ := res[0].(*big.Int)
		tokenTx = TokenTx{
			Sender : sender,
			To : to,
			Amount : amount,
			TokenAddress : event.Address,
			EventIndex : event.Index,
		}
		isTokenTx = true
		// fmt.Println(tokenTx, isTokenTx)
	}
	return tokenTx, isTokenTx
}

func InitTokenUserTxMap(block uint64, txExtractor TxExtractor, path string) {
	
	var err error
	proxy_tx_path := path + txExtractor.Dapp + "/historyTx/proxy/"
	files, err := ioutil.ReadDir(proxy_tx_path)
	if err != nil {
		fmt.Println(err)
	}
	tx_count := 0

	for _ , f := range files[:]{
		temp_list := strings.Split(strings.Split(f.Name(),".")[0], "_")
		b, err := strconv.ParseUint(temp_list[0],10,64)
		if err != nil {
			fmt.Println(err)
		}
		if b <= block{
			ExTx := vm.LoadTx(proxy_tx_path + f.Name())
			// if ExTx.BlockNumber.Cmp(big.NewInt(int64(11242635))) == 0{
			// 	ExTx.ParseTxTree()
			// }
			txExtractor.ExtractTokenTx(&ExTx)
			tx_count += 1
		} else {
			continue
		}
		// if tx_count > 200000{
		// 	break
		// }
	}
	fmt.Println("tx nums", tx_count)
}

func InitUserRelatedMapWithStake(block uint64, txExtractor TxExtractor, path string){

	var err error
	stake_tx_path := path + txExtractor.Dapp + "/historyTx/stake/"
	files, err := ioutil.ReadDir(stake_tx_path)
	if err != nil {
		fmt.Println(err)
	}
	for _ , f := range files{
		temp_list := strings.Split(strings.Split(f.Name(),".")[0], "_")
		b, err := strconv.ParseUint(temp_list[0],10,64)
		if err != nil {
			fmt.Println(err)
		}
		if b <= block{
			ExTx := vm.LoadTx(stake_tx_path + f.Name())
			txExtractor.ExtractStakeTokenTx(&ExTx)
		} else {
			break
		}
	}
}

func DeepCpUserTokenTxMap(userTokenTxMap UserTokenTxMap) UserTokenTxMap {
	var newUserTokenTxMap UserTokenTxMap
	newUserTokenTxMap = make(UserTokenTxMap)
	for userAddress := range(userTokenTxMap){
		newUserTokenTxMap[userAddress] = make(TokenTxMap)
		for tokenAddress := range userTokenTxMap[userAddress]{
			for _, tokenTx := range userTokenTxMap[userAddress][tokenAddress] {
				newUserTokenTxMap[userAddress][tokenAddress] = append(newUserTokenTxMap[userAddress][tokenAddress], tokenTx) 
			}
		}
	}

	return newUserTokenTxMap
}

func DeepCpUserRelatedMap(userRelatedMap UserRelatedMap) UserRelatedMap {
	var newUserRelatedMap UserRelatedMap
	newUserRelatedMap = make(UserRelatedMap)
	for userAddress := range userRelatedMap{
		newUserRelatedMap[userAddress] = make(RelatedMap)
		for relatedAddress := range userRelatedMap[userAddress]{
			newUserRelatedMap[userAddress][relatedAddress] = userRelatedMap[userAddress][relatedAddress]
		}
	}
	return newUserRelatedMap
}