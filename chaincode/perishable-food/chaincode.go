/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	_ "golang.org/x/crypto/bcrypt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// PerishableFood example
type PerishableFood struct {
}

// 账户
type Account struct {
	Id      string  `json:"id"`      // 账号ID
	Name    string  `json:"name"`    // 账号名
	Balance float64 `json:"balance"` // 余额
}

// 车位
type Commodity struct {
	Name     string  `json:"name"`     // 商品名
	Id       string  `json:"id"`       // 商品ID
	Location string  `json:"location"` // 地方
	Price    float64 `json:"price"`    // 费率
	OwnerId  string  `json:"owner"`    // 所有者
}

// 订单
type Order struct {
	Commodity *Commodity `json:"commodity"` //商品
	Id        string     `json:"id"`        //订单ID
	OrderTime time.Time  `json:"orderTime"` //下单时间
	Status    string     `json:"status"`    //订单状态
	BuyerId   string     `json:"buyer"`     //买家
	SellerId  string     `json:"seller"`    //卖家
}

// 订单状态
type Status struct {
	New        string // 新建
	Processing string // 运送中
	Done       string // 完成
	Canceled   string // 取消
}

// 状态枚举
func newStatus() *Status {
	return &Status{
		New:        "新建",
		Processing: "运送中",
		Done:       "完成",
		Canceled:   "取消",
	}
}

// 枚举键值对
var enumStatus = newStatus()
var statusMap = map[string]string{
	"New":        enumStatus.New,
	"Processing": enumStatus.Processing,
	"Done":       enumStatus.Done,
	"Canceled":   enumStatus.Canceled,
}

// 链码初始化
func (t *PerishableFood) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("链码初始化")

	// 初始化默认数据
	var names = [3]string{"国光", "红星", "红富士"}
	var ids = [3]string{
		"88efd7ea-bec6-4994-8ed1-f3f7b6f8cac7",
		"36bf5c7f-4cf7-4926-b0f6-0c5c18515752",
		"d9ce807b-e308-11e8-a47c-3e1591a6f5bb"}
	var accountsName = [3]string{"供货商", "物流商", "买家"}
	var accountList []string

	// 初始化账号数据，为“供应商”，“物流商”，“买家”账号初始化账号
	for i, val := range accountsName {
		account := &Account{
			Name:    val,
			Id:      strconv.Itoa(i + 1),
			Balance: 1000,
		}
		// 序列化对象
		bytes, err := json.Marshal(account)
		if err != nil {
			return shim.Error(fmt.Sprintf("marshal account error %s", err))
		}

		var key string
		if val, err := stub.CreateCompositeKey("account", []string{account.Id}); err != nil {
			return shim.Error(fmt.Sprintf("create key error %s", err))
		} else {
			key = val
		}

		if err := stub.PutState(key, bytes); err != nil {
			return shim.Error(fmt.Sprintf("put account error %s", err))
		}
		accountList = append(accountList, account.Id)
	}

	// 初始化商品数据，"国光", "红星", "红富士" 3种商品
	for i, val := range names {
		price := 6.00 + float64(i)
		commodity := &Commodity{
			Name:     val,
			Id:       ids[i],
			Location: "中国",
			//LowTemperature:  -2,
			//HighTemperature: 0,
			Price:   price, //单价
			OwnerId: accountList[0],
		}

		// 序列化对象
		commodityBytes, err := json.Marshal(commodity)
		if err != nil {
			return shim.Error(fmt.Sprintf("marshal commodity error %s", err))
		}

		var key string
		if val, err := stub.CreateCompositeKey("commodity", []string{commodity.Id}); err != nil {
			return shim.Error(fmt.Sprintf("create key error %s", err))
		} else {
			key = val
		}

		if err := stub.PutState(key, commodityBytes); err != nil {
			return shim.Error(fmt.Sprintf("put commodity error %s", err))
		}
	}

	return shim.Success(nil)
}

// 实现Invoke接口调用智能合约，例子中的所有智能合约都会集中在这个接口实现
func (t *PerishableFood) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	funcName, args := stub.GetFunctionAndParameters()

	switch funcName {
	// 创建商品ok/新建车位
	case "createCommodity":
		return createCommodity(stub, args)
	// 创建订单/发起出租
	case "createOrder":
		return createOrder(stub, args)
	// 查询商品列表ok
	case "queryCommodityList":
		return queryCommodityList(stub, args)
	// 查询订单列表
	case "queryOrderList":
		return queryOrderList(stub, args)
	// 查询账户ok
	case "queryAccount":
		return queryAccount(stub, args)
	// 更新订单状态
	case "updateOrderStatus":
		return updateOrderStatus(stub, args)
	default:
		return shim.Error(fmt.Sprintf("unsupported function: %s", funcName))
	}
}

// 新建商品
func createCommodity(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 检查参数的个数
	//if len(args) != 7 {
	if len(args) != 5 {
		return shim.Error("not enough args")
	}

	// 验证参数的正确性
	name := args[0]
	id := args[1]
	location := args[2]
	price := args[3]
	ownerId := args[4]

	//if name == "" || id == "" || location == "" || lowTemperature == "" || highTemperature == "" || price == "" || ownerId == "" {
	if name == "" || id == "" || location == "" || price == "" || ownerId == "" {
		return shim.Error("invalid args")
	}

	// 创建主键
	var key string
	if val, err := stub.CreateCompositeKey("commodity", []string{id}); err != nil {
		return shim.Error(fmt.Sprintf("create key error %s", err))
	} else {
		key = val
	}

	// 验证数据是否存在 应该存在 or 不应该存在
	if commodityBytes, err := stub.GetState(key); err == nil && len(commodityBytes) != 0 {
		return shim.Error("commodity already exist")
	}

	// 数据格式转换
	var formattedPrice float64
	if val, err := strconv.ParseFloat(price, 64); err != nil {
		return shim.Error("format price error")
	} else {
		formattedPrice = val
	}

	// 写入状态
	commodity := &Commodity{
		Name:     name,
		Id:       id,
		Location: location,
		Price:    formattedPrice, // 单价
		OwnerId:  ownerId,
	}

	// 序列化对象
	commodityBytes, err := json.Marshal(commodity)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal commodity error %s", err))
	}

	// 写入区块链账本
	if err := stub.PutState(key, commodityBytes); err != nil {
		return shim.Error(fmt.Sprintf("put commodity error %s", err))
	}

	// 成功返回
	return shim.Success(nil)
}

// 新建订单
func createOrder(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 检查参数的个数
	if len(args) != 6 {
		return shim.Error("not enough args")
	}

	// 验证参数的正确性
	commodityId := args[0]
	id := args[1]
	orderTime := args[2]
	status := args[3]
	buyerId := args[4]
	sellerId := args[5]

	//if commodityId == "" || id == "" || deliverAddress == "" || useTime == "" || quantity == "" || buyerId == "" || sellerId == "" || orderTime == "" {
	if commodityId == "" || id == "" || status == "" || buyerId == "" || sellerId == "" || orderTime == "" {
		return shim.Error("invalid args")
	}

	commodity := new(Commodity)
	// 验证数据是否存在 应该存在 or 不应该存在
	result, err := stub.GetStateByPartialCompositeKey("commodity", []string{commodityId})
	if err != nil {
		return shim.Error(fmt.Sprintf("Get commodity error %s", err))
	}
	defer result.Close()
	if result.HasNext() {
		val, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("Get commodity error %s", err))
		}

		if err := json.Unmarshal(val.GetValue(), commodity); err != nil {
			return shim.Error(fmt.Sprintf("Commodity failed to convert from bytes, error %s", err))
		}
	} else {
		return shim.Error("Commodity not exists")
	}

	// 数据格式转换
	var formattedOrderTime time.Time
	if val, err := time.Parse("2006-01-02 15:04:05", orderTime); err != nil {
		return shim.Error(fmt.Sprintf("formate useTime error: %s", err))
	} else {
		formattedOrderTime = val
	}

	// 写入状态
	order := &Order{
		Commodity: commodity,
		Id:        id,
		OrderTime: formattedOrderTime,
		Status:    enumStatus.New,
		BuyerId:   buyerId,
		SellerId:  sellerId,
	}

	// 序列化对象
	orderBytes, err := json.Marshal(order)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal order error %s", err))
	}

	// 创建主键
	var key string
	if val, err := stub.CreateCompositeKey("order", []string{id}); err != nil {
		return shim.Error(fmt.Sprintf("create key error %s", err))
	} else {
		key = val
	}

	if arr, err := stub.GetState(key); err != nil || len(arr) != 0 {
		fmt.Println(len(arr))
		return shim.Error("order already exists")
	}

	// 写入区块链账本
	if err := stub.PutState(key, orderBytes); err != nil {
		return shim.Error(fmt.Sprintf("put order error %s", err))
	}

	// 成功返回
	return shim.Success(nil)
}

// 查询商品列表
func queryCommodityList(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 检查参数的个数
	if len(args) > 1 {
		return shim.Error("no args required.")
	}

	keys := make([]string, 0)

	if len(args) == 1 {
		commodityId := args[0]
		if commodityId == "" {
			return shim.Error("invalid args")
		}
		keys = append(keys, commodityId)
	}
	// 通过主键从区块链查找相关的数据
	result, err := stub.GetStateByPartialCompositeKey("commodity", keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query commodity error: %s", err))
	}
	defer result.Close()

	// 检查返回的数据是否为空，不为空则遍历数据，否则返回空数组
	commoditylist := make([]*Commodity, 0)
	for result.HasNext() {
		val, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query commodity error: %s", err))
		}

		commodity := new(Commodity)
		if err := json.Unmarshal(val.GetValue(), commodity); err != nil {
			return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
		}

		commoditylist = append(commoditylist, commodity)
	}

	// 序列化数据
	bytes, err := json.Marshal(commoditylist)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal error: %s", err))
	}

	return shim.Success(bytes)
}

// 查询订单列表
func queryOrderList(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 检查参数的个数
	if len(args) > 1 {
		return shim.Error("no args required.")
	}

	keys := make([]string, 0)

	if len(args) == 1 {
		orderId := args[0]
		if orderId == "" {
			return shim.Error("invalid args")
		}
		keys = append(keys, orderId)
	}

	// 通过主键从区块链查找相关的数据
	result, err := stub.GetStateByPartialCompositeKey("order", keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query order error: %s", err))
	}
	defer result.Close()

	// 检查返回的数据是否为空，不为空则遍历数据，否则返回空数组
	orders := make([]*Order, 0)
	for result.HasNext() {
		val, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query orders error: %s", err))
		}

		order := new(Order)
		if err := json.Unmarshal(val.GetValue(), order); err != nil {
			return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
		}

		orders = append(orders, order)
	}

	// 序列化数据
	bytes, err := json.Marshal(orders)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal error: %s", err))
	}

	return shim.Success(bytes)
}

// 查询账号
func queryAccount(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args.")
	}

	// 验证参数的正确性
	accountId := args[0]

	if accountId == "" {
		return shim.Error("invalid args")
	}

	keys := make([]string, 0)
	if accountId != "all" {
		keys = append(keys, accountId)
	}

	// 通过主键从区块链查找相关的数据
	result, err := stub.GetStateByPartialCompositeKey("account", keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query account error: %s", err))
	}
	defer result.Close()

	// 检查返回的数据是否为空，不为空则遍历数据，否则返回空数组
	accounts := make([]*Account, 0)
	for result.HasNext() {
		val, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query accounts error: %s", err))
		}

		account := new(Account)
		if err := json.Unmarshal(val.GetValue(), account); err != nil {
			return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
		}

		accounts = append(accounts, account)
	}

	// 序列化数据
	bytes, err := json.Marshal(accounts)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal error: %s", err))
	}

	return shim.Success(bytes)
}

// 更新订单状态
func updateOrderStatus(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 检查参数的个数
	if len(args) != 2 {
		return shim.Error("not enough args.")
	}

	// 验证参数的正确性
	orderId := args[0]
	status := args[1]

	if orderId == "" || status == "" {
		return shim.Error("invalid args")
	}

	// 通过主键从区块链查找相关的数据
	keys := make([]string, 0)
	keys = append(keys, orderId)
	result, err := stub.GetStateByPartialCompositeKey("order", keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query order error: %s", err))
	}
	defer result.Close()

	// 检查返回的数据是否为空，不为空则遍历数据，否则返回空数组
	order := new(Order)
	for result.HasNext() {
		val, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query orders error: %s", err))
		}

		if err := json.Unmarshal(val.GetValue(), order); err != nil {
			return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
		}

		order.Status = statusMap[status]
	}

	// 订单完成状态的处理逻辑
	/**
	  若温度超出约定范围，将按如下公式进行计算。
	  没有超出范围将正常付款。
	  扣款计算公式：扣款 = 最低温度偏差值 * 0.1 * 货物数量 + 最高温度偏差值 * 0.2 * 货物数量
	*/
	if status == "Done" {
		accounts := make([]*Account, 0)
		// 获取买家账号
		buyerResult, buyErr := getStateByPartialCompositeKey(stub, order.BuyerId)
		if buyErr != nil {
			return shim.Error(fmt.Sprintf("query account error: %s", buyErr))
		}
		defer buyerResult.Close()

		for buyerResult.HasNext() {
			val, err := buyerResult.Next()
			if err != nil {
				return shim.Error(fmt.Sprintf("query accounts error: %s", err))
			}

			account := new(Account)
			if err := json.Unmarshal(val.GetValue(), account); err != nil {
				return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
			}

			accounts = append(accounts, account)
		}

		// 获取卖家账号
		sellerResult, sellerErr := getStateByPartialCompositeKey(stub, order.SellerId)
		if sellerErr != nil {
			return shim.Error(fmt.Sprintf("query account error: %s", sellerErr))
		}
		defer sellerResult.Close()

		for sellerResult.HasNext() {
			val, err := sellerResult.Next()
			if err != nil {
				return shim.Error(fmt.Sprintf("query accounts error: %s", err))
			}

			account := new(Account)
			if err := json.Unmarshal(val.GetValue(), account); err != nil {
				return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
			}

			accounts = append(accounts, account)
		}

		// 获取账号中的余额
		var buyerAcc *Account
		var sellerAcc *Account
		for _, val := range accounts {
			if val.Id == order.BuyerId {
				buyerAcc = val
			} else if val.Id == order.SellerId {
				sellerAcc = val
			}
		}
		var totalPrice float64
		now := time.Now()
		sumD := now.Sub(order.OrderTime)
		totalPrice = sumD.Hours() * order.Commodity.Price
		buyerAcc.Balance -= totalPrice
		sellerAcc.Balance += totalPrice

		// 序列化对象
		buyerBytes, sellerErr := json.Marshal(buyerAcc)
		if sellerErr != nil {
			return shim.Error(fmt.Sprintf("marshal buyer error %s", sellerErr))
		}

		var buyerKey string
		if val, err := stub.CreateCompositeKey("account", []string{order.BuyerId}); err != nil {
			return shim.Error(fmt.Sprintf("create key error %s", err))
		} else {
			buyerKey = val
		}

		if err := stub.PutState(buyerKey, buyerBytes); err != nil {
			return shim.Error(fmt.Sprintf("put buyer account error %s", err))
		}

		// 序列化对象
		sellerBytes, sellerErr := json.Marshal(sellerAcc)
		if sellerErr != nil {
			return shim.Error(fmt.Sprintf("marshal seller error %s", sellerErr))
		}

		var sellerKey string
		if val, err := stub.CreateCompositeKey("account", []string{order.SellerId}); err != nil {
			return shim.Error(fmt.Sprintf("create key error %s", err))
		} else {
			sellerKey = val
		}

		if err := stub.PutState(sellerKey, sellerBytes); err != nil {
			return shim.Error(fmt.Sprintf("put seller account error %s", err))
		}
	}

	// 序列化对象
	orderBytes, err := json.Marshal(order)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal order error %s", err))
	}

	// 构建主键
	var key string
	if val, err := stub.CreateCompositeKey("order", []string{orderId}); err != nil {
		return shim.Error(fmt.Sprintf("create key error %s", err))
	} else {
		key = val
	}

	// 写入区块链账本
	if err := stub.PutState(key, orderBytes); err != nil {
		return shim.Error(fmt.Sprintf("put commodity error %s", err))
	}

	return shim.Success(nil)
}

func getStateByPartialCompositeKey(stub shim.ChaincodeStubInterface, key string) (shim.StateQueryIteratorInterface, error) {
	keys := make([]string, 0)
	keys = append(keys, key)
	result, err := stub.GetStateByPartialCompositeKey("account", keys)
	return result, err
}

func main() {
	err := shim.Start(new(PerishableFood))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
