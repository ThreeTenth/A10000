package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocolID          = "/chat/1.0.0"
	discoveryServiceTag = "p2p-chat"
	rendezvousString    = "a10000-p2p-chat-rendezvous-point"
	bootstrapPeriod     = 5 * time.Minute
)

// 自定义网络参数
var (
	// 公共引导节点地址列表
	bootstrapPeers = []string{
		// IPFS公共引导节点，实际应用中可能需要配置自己的稳定节点
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	}
)

// mDNS 发现服务接口
type discoveryNotifee struct {
	h               host.Host
	ctx             context.Context
	activeStreams   map[peer.ID]network.Stream
	activeStreamsMu sync.Mutex
	userName        string
}

// 当通过 mDNS 发现新节点时处理
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.h.ID() {
		return // 忽略自己
	}

	fmt.Printf("\n通过 mDNS 发现节点: %s\n> ", pi.ID.String())
	n.connectToPeer(pi)
}

// 连接到新发现的节点
func (n *discoveryNotifee) connectToPeer(pi peer.AddrInfo) {
	// 将新发现的节点添加到对等节点存储
	n.h.Peerstore().AddAddrs(pi.ID, pi.Addrs, peerstore.PermanentAddrTTL)

	// 连接到新发现的节点
	err := n.h.Connect(n.ctx, pi)
	if err != nil {
		fmt.Printf("\n连接到节点 %s 失败: %s\n> ", pi.ID.String(), err)
		return
	}
	fmt.Printf("\n已连接到节点: %s\n> ", pi.ID.String())

	// 检查是否已有活跃的流
	n.activeStreamsMu.Lock()
	defer n.activeStreamsMu.Unlock()

	if _, exists := n.activeStreams[pi.ID]; !exists {
		// 打开一个新的流到刚连接的对等节点
		stream, err := n.h.NewStream(n.ctx, pi.ID, protocolID)
		if err != nil {
			fmt.Printf("\n无法打开流到 %s: %s\n> ", pi.ID.String(), err)
			return
		}

		// 保存这个流以便后续使用
		n.activeStreams[pi.ID] = stream

		// 处理这个流的输入
		handleStream(stream)

		// 发送欢迎消息
		welcomeMsg := fmt.Sprintf("[%s] 大家好！我已加入聊天。\n", n.userName)
		_, err = stream.Write([]byte(welcomeMsg))
		if err != nil {
			fmt.Printf("\n发送欢迎消息失败: %s\n> ", err)
		}
	}
}

// 启动 mDNS 发现服务（局域网发现）
func setupMDNSDiscovery(ctx context.Context, h host.Host, activeStreams map[peer.ID]network.Stream, userName string) error {
	// 创建一个发现服务通知器
	notifee := &discoveryNotifee{
		h:             h,
		ctx:           ctx,
		activeStreams: activeStreams,
		userName:      userName,
	}

	// 启动 mDNS 发现服务
	disc := mdns.NewMdnsService(h, discoveryServiceTag, notifee)
	return disc.Start()
}

// 启动 DHT 发现服务（广域网发现）
func setupDHTDiscovery(ctx context.Context, h host.Host, kadDHT *dht.IpfsDHT, activeStreams map[peer.ID]network.Stream, userName string) {
	// 创建一个发现服务通知器
	notifee := &discoveryNotifee{
		h:             h,
		ctx:           ctx,
		activeStreams: activeStreams,
		userName:      userName,
	}

	// 连接到引导节点
	go connectToBootstrapPeers(ctx, h)

	// 设置连接到 DHT 的自动重连
	go func() {
		for {
			// 通过 DHT 提供我们的节点信息
			fmt.Printf("\n在DHT中发布我们的节点信息 (使用rendezvous: %s)\n> ", rendezvousString)
			routingDiscovery := drouting.NewRoutingDiscovery(kadDHT)

			_, err := routingDiscovery.Advertise(ctx, rendezvousString)
			if err != nil {
				fmt.Printf("\nDHT广播失败: %s\n> ", err)
			}

			// 寻找其他使用相同 rendezvous 点的对等节点
			fmt.Printf("\n通过DHT寻找其他节点...\n> ")
			peerChan, err := routingDiscovery.FindPeers(ctx, rendezvousString)
			if err != nil {
				fmt.Printf("\n寻找对等节点失败: %s\n> ", err)
			} else {
				// 遍历发现的对等节点并连接
				for peer := range peerChan {
					if peer.ID == h.ID() {
						continue // 忽略自己
					}
					fmt.Printf("\n通过DHT发现节点: %s\n> ", peer.ID.String())
					notifee.connectToPeer(peer)
				}
			}

			// 定期重新广播/发现
			time.Sleep(bootstrapPeriod)
		}
	}()
}

// 连接到引导节点
func connectToBootstrapPeers(ctx context.Context, h host.Host) {
	// 解析引导节点地址
	var wg sync.WaitGroup
	for _, addr := range bootstrapPeers {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()

			maddr, err := multiaddr.NewMultiaddr(address)
			if err != nil {
				fmt.Printf("\n无效的引导节点地址: %s, %s\n> ", address, err)
				return
			}

			peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
			if err != nil {
				fmt.Printf("\n无法从地址解析对等节点信息: %s, %s\n> ", address, err)
				return
			}

			// 尝试连接到引导节点
			fmt.Printf("\n正在连接到引导节点: %s\n> ", peerInfo.ID.String())
			if err := h.Connect(ctx, *peerInfo); err != nil {
				fmt.Printf("\n连接到引导节点 %s 失败: %s\n> ", peerInfo.ID.String(), err)
				return
			}
			// fmt.Printf("\n已连接到引导节点: %s\n> ", peerInfo.ID.String())
			// notifee.connectToPeer(*peerInfo)
		}(addr)
	}
	wg.Wait()
}

// 创建一个支持 NAT 穿越和 DHT 的 libp2p 主机
func makeHost(ctx context.Context, port int, enableRelay bool) (host.Host, *dht.IpfsDHT, error) {
	// 生成随机密钥对或从配置文件加载
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 2048)
	if err != nil {
		return nil, nil, err
	}

	str, err := rcmgr.NewStatsTraceReporter()
	if err != nil {
		log.Fatal(err)
	}

	rmgr, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.DefaultLimits.AutoScale()), rcmgr.WithTraceReporter(str))
	if err != nil {
		log.Fatal(err)
	}

	// 创建连接管理器
	connManager, err := connmgr.NewConnManager(
		100, // 最小连接数
		400, // 最大连接数
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, nil, err
	}

	// 主机配置选项
	var opts []libp2p.Option

	// 基本配置
	opts = append(opts,
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),      // IPv4 TCP
			fmt.Sprintf("/ip6/::/tcp/%d", port),           // IPv6 TCP
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port), // IPv4 QUIC
			fmt.Sprintf("/ip6/::/udp/%d/quic", port),      // IPv6 QUIC
		),
		libp2p.ResourceManager(rmgr),
		libp2p.ConnectionManager(connManager),
		libp2p.DefaultSecurity,
		// libp2p.Security(noise.ID, noise.New),
		// libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.DefaultTransports,
		// libp2p.EnableAutoRelay(),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.NATPortMap(),
	)

	// 如果启用中继服务，添加中继选项
	if enableRelay {
		opts = append(opts, libp2p.EnableRelayService())
	}

	// 添加 DHT 路由
	var kadDHT *dht.IpfsDHT
	opts = append(opts, libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
		var err error
		kadDHT, err = dht.New(ctx, h, dht.Mode(dht.ModeAutoServer))
		return kadDHT, err
	}))

	// 使用选项创建主机
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, nil, err
	}

	return h, kadDHT, nil
}

// 处理传入的聊天消息
func handleStream(stream network.Stream) {
	// 创建一个缓冲读取器
	reader := bufio.NewReader(stream)

	go func() {
		for {
			// 从流中读取消息
			message, err := reader.ReadString('\n')
			if err != nil {
				peerID := stream.Conn().RemotePeer()
				fmt.Printf("\n用户 %s 已断开连接\n> ", peerID.String())
				stream.Close()
				return
			}

			// 显示接收到的消息
			fmt.Printf("\n%s> ", message)
		}
	}()
}

func main() {
	// 定义命令行参数
	listenPort := flag.Int("port", 0, "监听端口 (默认随机选择)")
	target := flag.String("connect", "", "要连接的对等节点 (可选, 格式: 地址/p2p/对等ID)")
	name := flag.String("name", "", "聊天名称")
	enableRelay := flag.Bool("relay", false, "作为中继节点运行")
	public := flag.Bool("public", false, "针对公共互联网使用优化选项")
	flag.Parse()

	// 检查聊天名称
	if *name == "" {
		*name = fmt.Sprintf("用户-%d", os.Getpid()%1000)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		http.Handle("/debug/metrics/prometheus", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":5001", nil))
	}()
	rcmgr.MustRegisterWith(prometheus.DefaultRegisterer)

	// 创建一个新的 libp2p 主机
	fmt.Println("初始化节点...")
	host, kadDHT, err := makeHost(ctx, *listenPort, *enableRelay)
	if err != nil {
		panic(err)
	}
	defer host.Close()

	// 存储活跃的流
	activeStreams := make(map[peer.ID]network.Stream)
	var activeStreamsMu sync.Mutex

	// 为聊天协议设置流处理程序
	host.SetStreamHandler(protocolID, func(stream network.Stream) {
		fmt.Printf("\n新的流连接来自: %s\n> ", stream.Conn().RemotePeer().String())

		// 保存这个流
		activeStreamsMu.Lock()
		activeStreams[stream.Conn().RemotePeer()] = stream
		activeStreamsMu.Unlock()

		// 处理这个流
		handleStream(stream)
	})

	// 打印主机地址和ID
	fmt.Println("节点ID:", host.ID().String())
	fmt.Println("监听地址:")
	for _, addr := range host.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), host.ID().String())
		fmt.Printf("- %s\n", fullAddr)
	}
	fmt.Printf("以 %s 身份进入聊天\n", *name)

	// 启动 DHT
	fmt.Println("启动 DHT...")
	if err := kadDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	// 启用局域网发现 (mDNS)
	fmt.Println("启动局域网发现服务 (mDNS)...")
	err = setupMDNSDiscovery(ctx, host, activeStreams, *name)
	if err != nil {
		panic(err)
	}

	// 如果开启了公共互联网模式，启用 DHT 发现
	if *public {
		fmt.Println("启动广域网发现服务 (DHT)...")
		setupDHTDiscovery(ctx, host, kadDHT, activeStreams, *name)
	}

	// 如果指定了目标地址，则连接到该对等节点
	if *target != "" {
		fmt.Println("正在连接到特定节点:", *target)

		// 解析目标多重地址
		targetAddr, err := multiaddr.NewMultiaddr(*target)
		if err != nil {
			panic(err)
		}

		// 从多重地址中提取对等节点信息
		peerInfo, err := peer.AddrInfoFromP2pAddr(targetAddr)
		if err != nil {
			panic(err)
		}

		// 将目标添加到对等节点存储，并设置长期连接保持时间
		host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)

		// 连接到目标对等节点
		err = host.Connect(ctx, *peerInfo)
		if err != nil {
			panic(err)
		}
		fmt.Println("已连接到", *target)

		// 打开一个流到目标对等节点
		stream, err := host.NewStream(ctx, peerInfo.ID, protocolID)
		if err != nil {
			panic(err)
		}

		// 保存这个流
		activeStreamsMu.Lock()
		activeStreams[peerInfo.ID] = stream
		activeStreamsMu.Unlock()

		// 设置流处理程序
		handleStream(stream)

		// 发送欢迎消息
		welcomeMsg := fmt.Sprintf("[%s] 大家好！我已加入聊天。\n", *name)
		_, err = stream.Write([]byte(welcomeMsg))
		if err != nil {
			panic(err)
		}
	}

	// 捕获信号以优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n正在优雅关闭...")

		// 发送告别消息
		goodbyeMsg := fmt.Sprintf("[%s] 我要离开聊天了，再见！\n", *name)
		activeStreamsMu.Lock()
		for peerID, stream := range activeStreams {
			_, err := stream.Write([]byte(goodbyeMsg))
			if err != nil {
				fmt.Printf("发送告别消息到 %s 失败: %s\n", peerID.String(), err)
			}
			stream.Close()
		}
		activeStreamsMu.Unlock()

		cancel()
		os.Exit(0)
	}()

	// 等待几秒钟，让发现机制启动
	fmt.Println("等待发现其他节点...")
	time.Sleep(2 * time.Second)

	// 开始聊天循环
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取输入时出错:", err)
			continue
		}

		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}

		// 处理命令
		if strings.HasPrefix(message, "/") {
			handleCommand(message, host, &activeStreamsMu, activeStreams, ctx)
			continue
		}

		// 发送消息到所有活跃的流
		formattedMsg := fmt.Sprintf("[%s] %s\n", *name, message)
		activeStreamsMu.Lock()
		for peerID, stream := range activeStreams {
			_, err = stream.Write([]byte(formattedMsg))
			if err != nil {
				fmt.Printf("\n发送消息到 %s 失败: %s\n> ", peerID.String(), err)
				// 如果发送失败，从活跃流中移除
				delete(activeStreams, peerID)
			}
		}
		activeStreamsMu.Unlock()
	}
}

// 处理用户命令
func handleCommand(cmd string, h host.Host, activeStreamsMu *sync.Mutex, activeStreams map[peer.ID]network.Stream, ctx context.Context) {
	switch cmd {
	case "/quit":
		fmt.Println("退出聊天...")
		os.Exit(0)

	case "/peers":
		// 显示已连接的对等节点
		fmt.Println("已连接的对等节点:")
		activeStreamsMu.Lock()
		for peerID := range activeStreams {
			fmt.Printf("- %s\n", peerID.String())
		}
		activeStreamsMu.Unlock()

	case "/addr":
		// 显示本节点的地址
		fmt.Println("本节点地址:")
		for _, addr := range h.Addrs() {
			fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), h.ID().String())
			fmt.Printf("- %s\n", fullAddr)
		}

	case "/dht":
		// 显示DHT信息
		fmt.Println("正在寻找更多节点...")
		peers := h.Network().Peers()
		fmt.Printf("当前网络中有 %d 个节点\n", len(peers))

	case "/help":
		// 显示帮助信息
		fmt.Println("可用命令:")
		fmt.Println("  /quit - 退出聊天")
		fmt.Println("  /peers - 显示已连接的对等节点")
		fmt.Println("  /addr - 显示本节点的地址")
		fmt.Println("  /dht - 显示DHT信息")
		fmt.Println("  /help - 显示此帮助信息")

	default:
		if strings.HasPrefix(cmd, "/connect ") {
			// 连接到指定节点
			addr := strings.TrimPrefix(cmd, "/connect ")
			targetAddr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				fmt.Printf("无效的地址格式: %s\n", err)
				return
			}

			peerInfo, err := peer.AddrInfoFromP2pAddr(targetAddr)
			if err != nil {
				fmt.Printf("无法解析对等节点信息: %s\n", err)
				return
			}

			fmt.Printf("正在连接到 %s...\n", peerInfo.ID.String())
			err = h.Connect(ctx, *peerInfo)
			if err != nil {
				fmt.Printf("连接失败: %s\n", err)
				return
			}

			// 打开流
			stream, err := h.NewStream(ctx, peerInfo.ID, protocolID)
			if err != nil {
				fmt.Printf("无法打开流: %s\n", err)
				return
			}

			// 保存流
			activeStreamsMu.Lock()
			activeStreams[peerInfo.ID] = stream
			activeStreamsMu.Unlock()

			// 处理流
			handleStream(stream)
			fmt.Printf("已连接到 %s\n", peerInfo.ID.String())
		} else {
			fmt.Println("未知命令。输入 /help 获取帮助。")
		}
	}
}
