package market

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"nofx/hook"
	"strconv"
	"time"
)

const (
	baseURL = "https://fapi.binance.com"
)

type APIClient struct {
	client *http.Client
}

func NewAPIClient() *APIClient {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	hookRes := hook.HookExec[hook.SetHttpClientResult](hook.SET_HTTP_CLIENT, client)
	if hookRes != nil && hookRes.Error() == nil {
		log.Printf("使用Hook设置的HTTP客户端")
		client = hookRes.GetResult()
	}

	return &APIClient{
		client: client,
	}
}

func (c *APIClient) GetExchangeInfo() (*ExchangeInfo, error) {
	url := fmt.Sprintf("%s/fapi/v1/exchangeInfo", baseURL)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var exchangeInfo ExchangeInfo
	err = json.Unmarshal(body, &exchangeInfo)
	if err != nil {
		return nil, err
	}

	return &exchangeInfo, nil
}

func (c *APIClient) GetKlines(symbol, interval string, limit int) ([]Kline, error) {
	url := fmt.Sprintf("%s/fapi/v1/klines", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("symbol", symbol)
	q.Add("interval", interval)
	q.Add("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var klineResponses []KlineResponse
	err = json.Unmarshal(body, &klineResponses)
	if err != nil {
		log.Printf("获取K线数据失败,响应内容: %s", string(body))
		return nil, err
	}

	var klines []Kline
	for _, kr := range klineResponses {
		kline, err := parseKline(kr)
		if err != nil {
			log.Printf("解析K线数据失败: %v", err)
			continue
		}
		klines = append(klines, kline)
	}

	return klines, nil
}

func parseKline(kr KlineResponse) (Kline, error) {
	var kline Kline

	if len(kr) < 11 {
		return kline, fmt.Errorf("invalid kline data")
	}

	// 解析各个字段
	kline.OpenTime = int64(kr[0].(float64))
	kline.Open, _ = strconv.ParseFloat(kr[1].(string), 64)
	kline.High, _ = strconv.ParseFloat(kr[2].(string), 64)
	kline.Low, _ = strconv.ParseFloat(kr[3].(string), 64)
	kline.Close, _ = strconv.ParseFloat(kr[4].(string), 64)
	kline.Volume, _ = strconv.ParseFloat(kr[5].(string), 64)
	kline.CloseTime = int64(kr[6].(float64))
	kline.QuoteVolume, _ = strconv.ParseFloat(kr[7].(string), 64)
	kline.Trades = int(kr[8].(float64))
	kline.TakerBuyBaseVolume, _ = strconv.ParseFloat(kr[9].(string), 64)
	kline.TakerBuyQuoteVolume, _ = strconv.ParseFloat(kr[10].(string), 64)

	return kline, nil
}

func (c *APIClient) GetCurrentPrice(symbol string) (float64, error) {
	url := fmt.Sprintf("%s/fapi/v1/ticker/price", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	q := req.URL.Query()
	q.Add("symbol", symbol)
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var ticker PriceTicker
	err = json.Unmarshal(body, &ticker)
	if err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(ticker.Price, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

// GetKlinesRange 获取指定时间范围的K线数据(支持大批量下载)
// Binance API限制每次最多返回1500条K线,该方法会自动分批请求
func (c *APIClient) GetKlinesRange(symbol, interval string, startTime, endTime int64) ([]Kline, error) {
	const maxLimit = 1500 // Binance API限制
	var allKlines []Kline

	// 计算间隔毫秒数
	intervalMs, err := parseIntervalToMs(interval)
	if err != nil {
		return nil, fmt.Errorf("invalid interval: %w", err)
	}

	currentStart := startTime
	for currentStart < endTime {
		// 计算本次请求的结束时间(不超过maxLimit条)
		maxEnd := currentStart + int64(maxLimit)*intervalMs
		currentEnd := endTime
		if maxEnd < endTime {
			currentEnd = maxEnd
		}

		// 请求本批数据
		klines, err := c.getKlinesBatch(symbol, interval, currentStart, currentEnd, maxLimit)
		if err != nil {
			return nil, fmt.Errorf("failed to get klines batch: %w", err)
		}

		if len(klines) == 0 {
			break
		}

		allKlines = append(allKlines, klines...)

		// 更新下次请求的起始时间(使用最后一条K线的closeTime + 1)
		currentStart = klines[len(klines)-1].CloseTime + 1

		// 避免API限流(每50ms一次请求)
		time.Sleep(50 * time.Millisecond)
	}

	return allKlines, nil
}

// getKlinesBatch 获取单批K线数据
func (c *APIClient) getKlinesBatch(symbol, interval string, startTime, endTime int64, limit int) ([]Kline, error) {
	url := fmt.Sprintf("%s/fapi/v1/klines", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("symbol", symbol)
	q.Add("interval", interval)
	q.Add("startTime", strconv.FormatInt(startTime, 10))
	q.Add("endTime", strconv.FormatInt(endTime, 10))
	q.Add("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var klineResponses []KlineResponse
	err = json.Unmarshal(body, &klineResponses)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal klines: %w", err)
	}

	var klines []Kline
	for _, kr := range klineResponses {
		kline, err := parseKline(kr)
		if err != nil {
			log.Printf("解析K线数据失败: %v", err)
			continue
		}
		klines = append(klines, kline)
	}

	return klines, nil
}

// parseIntervalToMs 将间隔字符串转换为毫秒
func parseIntervalToMs(interval string) (int64, error) {
	if len(interval) < 2 {
		return 0, fmt.Errorf("invalid interval format: %s", interval)
	}

	unit := interval[len(interval)-1]
	valueStr := interval[:len(interval)-1]
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid interval value: %s", interval)
	}

	switch unit {
	case 'm':
		return value * 60 * 1000, nil
	case 'h':
		return value * 60 * 60 * 1000, nil
	case 'd':
		return value * 24 * 60 * 60 * 1000, nil
	default:
		return 0, fmt.Errorf("unsupported interval unit: %c", unit)
	}
}
