package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"

	"counter"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// 1. 配置连接
	// 替换为你的 Sepolia RPC URL (例如 Alchemy 或 Infura)
	rpcURL := os.Getenv("SEPOLIA_URL")
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("无法连接到以太坊节点: %v", err)
	}
	fmt.Println("已成功连接到 Sepolia 测试网")

	wsrpcURL := os.Getenv("ETH_WS_URL")
	if wsrpcURL == "" {
		wsrpcURL = os.Getenv("ETH_RPC_URL")
	}
	if wsrpcURL == "" {
		log.Fatal("ETH_WS_URL or ETH_RPC_URL must be set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wsclient, err := ethclient.DialContext(ctx, wsrpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer wsclient.Close()
	fmt.Println("已成功连接到 Sepolia WebSockets")

	var tx *types.Transaction

	// 2. 设置账户
	// 从环境变量读取私钥，避免硬编码在代码中
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatal("请设置 PRIVATE_KEY 环境变量")
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("私钥解析失败: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("无法转换公钥类型")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// 获取 nonce 和 gas price
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("获取 nonce 失败: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("获取 gas price 失败: %v", err)
	}

	// 创建交易签名者
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(11155111))
	if err != nil {
		log.Fatalf("创建交易签名者失败: %v", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // 转账金额 (wei)
	auth.GasLimit = uint64(3000000) // Gas 限制
	auth.GasPrice = gasPrice

	// 3. 部署合约 (如果合约已经部署在 Sepolia 上，可以跳过此步，直接使用已知地址)

	// fmt.Println("正在部署合约...")
	// address, tx, instance, err := counter.DeployCounter(auth, client, big.NewInt(0))
	// if err != nil {
	// 	log.Fatalf("合约部署失败: %v", err)
	// }
	// fmt.Printf("合约部署交易中: %s\n", tx.Hash().Hex())
	// fmt.Printf("等待合约部署确认 (地址: %s) ...\n", address.Hex())

	// // 等待交易被打包
	// ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	// defer cancel()
	// deployAddr, err := bind.WaitDeployed(ctx, client, tx)
	// if err != nil {
	// 	log.Fatalf("等待部署确认失败: %v", err)
	// }
	// fmt.Printf("合约部署成功! 地址: %s\n", deployAddr.Hex())

	// 如果合约已存在，使用以下代码加载实例：用 HTTP client 创建读写实例（调用 / 交易）
	contractAddress := common.HexToAddress("0x6f17C320Bea20d69Fb16fbca7299afce2BDfA4A6")
	instance, err := counter.NewCounter(contractAddress, client)

	// 专门用 wsclient 创建 Filterer，用来订阅事件
	filterer, err := counter.NewCounterFilterer(contractAddress, wsclient)
	if err != nil {
		log.Fatalf("创建事件过滤器失败: %v", err)
	}

	// 4. 与合约交互

	// A. 读取初始值 (Call 操作，不需要 Gas，不需要私钥签名)
	count, err := instance.GetCount(&bind.CallOpts{})
	if err != nil {
		log.Fatalf("获取计数失败: %v", err)
	}
	fmt.Printf("当前计数值: %s\n", count.String())

	// 事件通道
	events := make(chan *counter.CounterCountChanged)

	// 订阅 CountChanged 事件（不过滤 caller，传空切片）
	sub, err := filterer.WatchCountChanged(
		&bind.WatchOpts{
			Context: context.Background(),
			// Start 留空表示从当前最新块开始监听
		},
		events,
		[]common.Address{}, // caller 过滤条件，空表示不过滤
	)
	if err != nil {
		log.Fatalf("订阅 CountChanged 事件失败: %v", err)
	}
	defer sub.Unsubscribe()

	// 起一个 goroutine 处理事件
	go func() {
		for {
			select {
			case ev := <-events:
				fmt.Printf(
					"收到 CountChanged 事件: caller=%s newValue=%s action=%s\n",
					ev.Caller.Hex(),
					ev.NewValue.String(),
					ev.Action,
				)
			case err := <-sub.Err():
				log.Printf("事件订阅出错: %v\n", err)
				return
			}
		}
	}()

	// B. 增加计数 (Transaction 操作，需要 Gas 和签名)
	fmt.Println("正在调用 increment() 增加计数...")
	tx, err = instance.Increment(auth)
	if err != nil {
		log.Fatalf("调用 increment 失败: %v", err)
	}
	fmt.Printf("交易发送成功: %s\n", tx.Hash().Hex())

	// 等待交易确认
	fmt.Println("等待交易确认...")
	mineReceipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatalf("等待交易确认失败: %v", err)
	}
	fmt.Printf("交易已确认! Block: %d, Status: %d\n", mineReceipt.BlockNumber, mineReceipt.Status)

	// 查询当前交易所在区块及之后的 CountChanged 历史事件
	iter, err := filterer.FilterCountChanged(
		&bind.FilterOpts{
			Start:   mineReceipt.BlockNumber.Uint64(),
			End:     nil,
			Context: context.Background(),
		},
		[]common.Address{},
	)
	if err != nil {
		log.Fatalf("过滤 CountChanged 失败: %v", err)
	}
	defer iter.Close()

	for iter.Next() {
		ev := iter.Event
		fmt.Printf("历史 CountChanged: caller=%s newValue=%s action=%s\n",
			ev.Caller.Hex(), ev.NewValue.String(), ev.Action)
	}
	if err := iter.Error(); err != nil {
		log.Fatalf("遍历事件出错: %v", err)
	}

	// C. 再次读取值
	newCount, err := instance.GetCount(&bind.CallOpts{})
	if err != nil {
		log.Fatalf("获取新计数失败: %v", err)
	}
	fmt.Printf("新的计数值: %s\n", newCount.String())

	if newCount.Cmp(count) == 1 {
		fmt.Println("✅ 成功：计数器已增加！")
	} else {
		fmt.Println("❌ 失败：计数器未变化。")
	}
}
