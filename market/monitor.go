package market

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type WSMonitor struct {
	wsClient       *WSClient
	combinedClient *CombinedStreamsClient
	symbols        []string
	featuresMap    sync.Map
	alertsChan     chan Alert
	klineDataMaps  map[string]*sync.Map // 动态存储所有时间框架的K线数据: "3m" -> sync.Map, "1h" -> sync.Map, etc.
	tickerDataMap  sync.Map             // 存储每个交易对的ticker数据
	batchSize      int
	filterSymbols  sync.Map     // 使用sync.Map来存储需要监控的币种和其状态
	symbolStats    sync.Map     // 存储币种统计信息
	FilterSymbol   []string     //经过筛选的币种
	klineMapsMutex sync.RWMutex // 保护klineDataMaps的并发访问
}
type SymbolStats struct {
	LastActiveTime   time.Time
	AlertCount       int
	VolumeSpikeCount int
	LastAlertTime    time.Time
	Score            float64 // 综合评分
}

var WSMonitorCli *WSMonitor
var subKlineTime = []string{"3m", "4h"} // 管理订阅流的K线周期

func NewWSMonitor(batchSize int) *WSMonitor {
	// 初始化所有支持的时间框架的数据存储
	klineDataMaps := make(map[string]*sync.Map)
	supportedTimeframes := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "1d"}
	for _, tf := range supportedTimeframes {
		klineDataMaps[tf] = &sync.Map{}
	}

	WSMonitorCli = &WSMonitor{
		wsClient:       NewWSClient(),
		combinedClient: NewCombinedStreamsClient(batchSize),
		alertsChan:     make(chan Alert, 1000),
		batchSize:      batchSize,
		klineDataMaps:  klineDataMaps,
	}
	return WSMonitorCli
}

func (m *WSMonitor) Initialize(coins []string) error {
	log.Println("初始化WebSocket监控器...")

	// 记录支持的所有时间框架
	supportedTimeframes := []string{}
	m.klineMapsMutex.RLock()
	for tf := range m.klineDataMaps {
		supportedTimeframes = append(supportedTimeframes, tf)
	}
	m.klineMapsMutex.RUnlock()
	log.Printf("✅ 支持的时间框架: %v", supportedTimeframes)

	// 获取交易对信息
	apiClient := NewAPIClient()
	// 如果不指定交易对，则使用market市场的所有交易对币种
	if len(coins) == 0 {
		exchangeInfo, err := apiClient.GetExchangeInfo()
		if err != nil {
			return err
		}
		// 筛选永续合约交易对 --仅测试时使用
		//exchangeInfo.Symbols = exchangeInfo.Symbols[0:2]
		for _, symbol := range exchangeInfo.Symbols {
			if symbol.Status == "TRADING" && symbol.ContractType == "PERPETUAL" && strings.ToUpper(symbol.Symbol[len(symbol.Symbol)-4:]) == "USDT" {
				m.symbols = append(m.symbols, symbol.Symbol)
				m.filterSymbols.Store(symbol.Symbol, true)
			}
		}
	} else {
		m.symbols = coins
	}

	log.Printf("找到 %d 个交易对", len(m.symbols))
	// 初始化历史数据
	if err := m.initializeHistoricalData(); err != nil {
		log.Printf("初始化历史数据失败: %v", err)
	}

	return nil
}

func (m *WSMonitor) initializeHistoricalData() error {
	apiClient := NewAPIClient()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // 限制并发数

	for _, symbol := range m.symbols {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(s string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// 获取历史K线数据 - 3m
			klines, err := apiClient.GetKlines(s, "3m", 100)
			if err != nil {
				log.Printf("获取 %s 历史数据失败: %v", s, err)
				return
			}
			if len(klines) > 0 {
				if klineMap3m := m.getKlineDataMap("3m"); klineMap3m != nil {
					klineMap3m.Store(s, klines)
					log.Printf("已加载 %s 的历史K线数据-3m: %d 条", s, len(klines))
				}
			}

			// 获取历史K线数据 - 4h
			klines4h, err := apiClient.GetKlines(s, "4h", 100)
			if err != nil {
				log.Printf("获取 %s 历史数据失败: %v", s, err)
				return
			}
			if len(klines4h) > 0 {
				if klineMap4h := m.getKlineDataMap("4h"); klineMap4h != nil {
					klineMap4h.Store(s, klines4h)
					log.Printf("已加载 %s 的历史K线数据-4h: %d 条", s, len(klines4h))
				}
			}
		}(symbol)
	}

	wg.Wait()
	return nil
}

func (m *WSMonitor) Start(coins []string) {
	log.Printf("启动WebSocket实时监控...")
	// 初始化交易对
	err := m.Initialize(coins)
	if err != nil {
		log.Printf("❌ 初始化币种失败: %v", err)
		return
	}

	err = m.combinedClient.Connect()
	if err != nil {
		log.Printf("⚠️ 实时数据流连接失败，将仅使用历史数据: %v", err)
		log.Printf("💡 系统将继续运行，AI决策基于历史K线数据")
		// 不返回错误，允许系统继续运行
		return
	}
	// 订阅所有交易对
	err = m.subscribeAll()
	if err != nil {
		log.Printf("⚠️ 订阅币种交易对失败: %v", err)
		log.Printf("💡 系统将继续运行，AI决策基于历史K线数据")
		// 不返回错误，允许系统继续运行
		return
	}
	log.Printf("✅ 实时数据流订阅成功")
}

// subscribeSymbol 注册监听
func (m *WSMonitor) subscribeSymbol(symbol, st string) []string {
	var streams []string
	stream := fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), st)
	ch := m.combinedClient.AddSubscriber(stream, 100)
	streams = append(streams, stream)
	go m.handleKlineData(symbol, ch, st)

	return streams
}
func (m *WSMonitor) subscribeAll() error {
	// 执行批量订阅
	log.Println("开始订阅所有交易对...")
	for _, symbol := range m.symbols {
		for _, st := range subKlineTime {
			m.subscribeSymbol(symbol, st)
		}
	}
	for _, st := range subKlineTime {
		err := m.combinedClient.BatchSubscribeKlines(m.symbols, st)
		if err != nil {
			log.Printf("❌ 订阅 %s K线失败: %v", st, err)
			return err
		}
	}
	log.Println("所有交易对订阅完成")
	return nil
}

func (m *WSMonitor) handleKlineData(symbol string, ch <-chan []byte, _time string) {
	for data := range ch {
		var klineData KlineWSData
		if err := json.Unmarshal(data, &klineData); err != nil {
			log.Printf("解析Kline数据失败: %v", err)
			continue
		}
		m.processKlineUpdate(symbol, klineData, _time)
	}
}

func (m *WSMonitor) getKlineDataMap(_time string) *sync.Map {
	m.klineMapsMutex.RLock()
	defer m.klineMapsMutex.RUnlock()

	if klineMap, exists := m.klineDataMaps[_time]; exists {
		return klineMap
	}

	// 如果时间框架不存在,记录警告并返回nil
	log.Printf("警告: 不支持的时间框架 %s,请使用以下之一: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 12h, 1d", _time)
	return nil
}
func (m *WSMonitor) processKlineUpdate(symbol string, wsData KlineWSData, _time string) {
	// 转换WebSocket数据为Kline结构
	kline := Kline{
		OpenTime:  wsData.Kline.StartTime,
		CloseTime: wsData.Kline.CloseTime,
		Trades:    wsData.Kline.NumberOfTrades,
	}
	kline.Open, _ = parseFloat(wsData.Kline.OpenPrice)
	kline.High, _ = parseFloat(wsData.Kline.HighPrice)
	kline.Low, _ = parseFloat(wsData.Kline.LowPrice)
	kline.Close, _ = parseFloat(wsData.Kline.ClosePrice)
	kline.Volume, _ = parseFloat(wsData.Kline.Volume)
	kline.High, _ = parseFloat(wsData.Kline.HighPrice)
	kline.QuoteVolume, _ = parseFloat(wsData.Kline.QuoteVolume)
	kline.TakerBuyBaseVolume, _ = parseFloat(wsData.Kline.TakerBuyBaseVolume)
	kline.TakerBuyQuoteVolume, _ = parseFloat(wsData.Kline.TakerBuyQuoteVolume)

	// 更新K线数据
	var klineDataMap = m.getKlineDataMap(_time)
	if klineDataMap == nil {
		log.Printf("警告: processKlineUpdate 收到不支持的时间框架 %s,忽略更新", _time)
		return
	}

	value, exists := klineDataMap.Load(symbol)
	var klines []Kline
	if exists {
		klines = value.([]Kline)

		// 检查是否是新的K线
		if len(klines) > 0 && klines[len(klines)-1].OpenTime == kline.OpenTime {
			// 更新当前K线
			klines[len(klines)-1] = kline
		} else {
			// 添加新K线
			klines = append(klines, kline)

			// 保持数据长度
			if len(klines) > 100 {
				klines = klines[1:]
			}
		}
	} else {
		klines = []Kline{kline}
	}

	klineDataMap.Store(symbol, klines)
}

func (m *WSMonitor) GetCurrentKlines(symbol string, duration string) ([]Kline, error) {
	// 获取对应时间框架的数据映射
	klineDataMap := m.getKlineDataMap(duration)
	if klineDataMap == nil {
		return nil, fmt.Errorf("不支持的时间框架: %s", duration)
	}

	// 对每一个进来的symbol检测是否存在内类 是否的话就订阅它
	value, exists := klineDataMap.Load(symbol)
	if !exists {
		// 如果Ws数据未初始化完成时,单独使用api获取 - 兼容性代码 (防止在未初始化完成是,已经有交易员运行)
		apiClient := NewAPIClient()
		klines, err := apiClient.GetKlines(symbol, duration, 100)
		if err != nil {
			return nil, fmt.Errorf("获取%v K线失败: %v", duration, err)
		}

		// 动态缓存进缓存
		klineDataMap.Store(strings.ToUpper(symbol), klines)

		// 订阅 WebSocket 流
		subStr := m.subscribeSymbol(symbol, duration)
		subErr := m.combinedClient.subscribeStreams(subStr)
		log.Printf("动态订阅流: %v (时间框架: %s)", subStr, duration)
		if subErr != nil {
			log.Printf("警告: 动态订阅%v K线失败: %v (使用API数据)", duration, subErr)
		}


		// ✅ FIX: 返回深拷贝而非引用
		result := make([]Kline, len(klines))
		copy(result, klines)
		return result, nil
	}

	// ✅ FIX: 返回深拷贝而非引用，避免并发竞态条件
	klines := value.([]Kline)
	result := make([]Kline, len(klines))
	copy(result, klines)
	return result, nil
}

func (m *WSMonitor) Close() {
	m.wsClient.Close()
	close(m.alertsChan)
}
