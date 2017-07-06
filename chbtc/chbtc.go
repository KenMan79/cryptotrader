// Package chbtc CHBTC rest api package
package chbtc

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/Akagi201/cryptotrader/model"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	MarketAPI = "http://api.chbtc.com/data/v1/"
	TradeAPI  = "https://trade.chbtc.com/api/"
)

type CHBTC struct {
	AccessKey string
	SecretKey string
}

func New(accessKey string, secretKey string) *CHBTC {
	return &CHBTC{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}
}

// GetTicker 行情
func (cb *CHBTC) GetTicker(base string, quote string) (*model.Ticker, error) {
	log.Debugf("Currency base: %s, quote: %s", base, quote)

	url := MarketAPI + "ticker?currency=" + quote + "_" + base

	log.Debugf("Request url: %v", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Debugf("Response body: %v", string(body))

	buyRes := gjson.GetBytes(body, "ticker.buy").String()
	buy, err := strconv.ParseFloat(buyRes, 64)
	if err != nil {
		return nil, err
	}

	sellRes := gjson.GetBytes(body, "ticker.sell").String()
	sell, err := strconv.ParseFloat(sellRes, 64)
	if err != nil {
		return nil, err
	}

	lastRes := gjson.GetBytes(body, "ticker.last").String()
	last, err := strconv.ParseFloat(lastRes, 64)
	if err != nil {
		return nil, err
	}

	lowRes := gjson.GetBytes(body, "ticker.low").String()
	low, err := strconv.ParseFloat(lowRes, 64)
	if err != nil {
		return nil, err
	}

	highRes := gjson.GetBytes(body, "ticker.high").String()
	high, err := strconv.ParseFloat(highRes, 64)
	if err != nil {
		return nil, err
	}

	volRes := gjson.GetBytes(body, "ticker.vol").String()
	vol, err := strconv.ParseFloat(volRes, 64)
	if err != nil {
		return nil, err
	}

	return &model.Ticker{
		Buy:  buy,
		Sell: sell,
		Last: last,
		Low:  low,
		High: high,
		Vol:  vol,
	}, nil
}

// GetOrderBook 市场深度
// size: 档位 1-50, 如果有合并深度, 只能返回 5 档深度
// merge:
// btc_cny: 可选 1, 0.1
// ltc_cny: 可选 0.5, 0.3, 0.1
// eth_cny: 可选 0.5, 0.3, 0.1
// etc_cny: 可选 0.3, 0.1
// bts_cny: 可选 1, 0.1
func (cb *CHBTC) GetOrderBook(base string, quote string, size int, merge float64) (*model.OrderBook, error) {
	url := MarketAPI + "depth?currency=" + quote + "_" + base + "&size=" + strconv.Itoa(size) + "&merge=" + strconv.FormatFloat(merge, 'f', -1, 64)

	log.Debugf("Request url: %v", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Debugf("Response body: %v", string(body))

	orderBook := &model.OrderBook{
		Base:  base,
		Quote: quote,
		Time:  time.Unix(gjson.GetBytes(body, "timestamp").Int(), 0),
	}

	gjson.GetBytes(body, "asks").ForEach(func(k, v gjson.Result) bool {
		orderBook.Asks = append(orderBook.Asks, &model.Order{
			Price:  v.Array()[0].Float(),
			Amount: v.Array()[1].Float(),
		})

		return true
	})

	gjson.GetBytes(body, "bids").ForEach(func(k, v gjson.Result) bool {
		orderBook.Bids = append(orderBook.Bids, &model.Order{
			Price:  v.Array()[0].Float(),
			Amount: v.Array()[1].Float(),
		})

		return true
	})

	return orderBook, nil
}

// GetTrades 获取历史成交
// currency: quote_base
// btc_cny: 比特币/人民币
// ltc_cny: 莱特币/人民币
// eth_cny: 以太币/人民币
// etc_cny: ETC币/人民币
// bts_cny: BTS币/人民币
// since: 从指定交易 ID 后 50 条数据
func (cb *CHBTC) GetTrades(base string, quote string, since int) (*model.Trades, error) {
	url := MarketAPI + "trades?currency=" + quote + "_" + base
	if since != 0 {
		url += "&since=" + strconv.Itoa(since)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Debugf("Response body: %v", string(body))

	trades := new(model.Trades)

	gjson.ParseBytes(body).ForEach(func(k, v gjson.Result) bool {
		trade := &model.Trade{
			Amount:    v.Get("amount").Float(),
			Price:     v.Get("price").Float(),
			Tid:       v.Get("tid").Int(),
			TradeType: v.Get("trade_type").String(),
			Type:      v.Get("type").String(),
			Date:      time.Unix(v.Get("date").Int(), 0),
		}
		*trades = append(*trades, trade)
		return true
	})

	return trades, nil
}

// GetKline 获取 K 线数据
// currency: quote_base
// btc_cny: 比特币/人民币
// ltc_cny: 莱特币/人民币
// eth_cny: 以太币/人民币
// etc_cny: ETC币/人民币
// bts_cny: BTS币/人民币
// typ:
// 1min: 1 分钟
// 3min: 3 分钟
// 5min: 5 分钟
// 15min: 15 分钟
// 30min: 30 分钟
// 1day: 1 日
// 3day: 3 日
// 1week: 1 周
// 1hour: 1 小时
// 2hour: 2 小时
// 4hour: 4 小时
// 6hour: 6小时
// 12hour: 12 小时
// since: 从这个时间戳之后的
// size: 返回数据的条数限制(默认为 1000, 如果返回数据多于 1000 条, 那么只返回 1000 条)
func (cb *CHBTC) GetKline(base string, quote string, typ string, since int, size int) (*model.Kline, error) {
	url := MarketAPI + "kline?currency=" + quote + "_" + base

	if len(typ) != 0 {
		url += "&type=" + typ
	}

	if since != 0 {
		url += "&since=" + strconv.Itoa(since)
	}

	if size != 0 {
		url += "&size=" + strconv.Itoa(size)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Debugf("Response body: %v", string(body))

	kline := new(model.Kline)

	kline.MoneyType = gjson.GetBytes(body, "moneyType").String()
	kline.Symbol = gjson.GetBytes(body, "symbol").String()

	gjson.GetBytes(body, "data").ForEach(func(k, v gjson.Result) bool {
		klinedata := &model.KlineData{
			Time:   time.Unix(v.Array()[0].Int()/1000, 0),
			Open:   v.Array()[1].Float(),
			High:   v.Array()[2].Float(),
			Low:    v.Array()[3].Float(),
			Close:  v.Array()[4].Float(),
			Amount: v.Array()[5].Float(),
		}

		kline.Data = append(kline.Data, klinedata)
		return true
	})

	return kline, nil
}

// SecretDigest calc secert digest
func (cb *CHBTC) SecretDigest() string {
	sha := sha1.New()
	sha.Write([]byte(cb.SecretKey))
	return hex.EncodeToString(sha.Sum(nil))
}

// Sign calc sign string
func (cb *CHBTC) Sign(uri string) string {
	digest := cb.SecretDigest()
	mac := hmac.New(md5.New, []byte(digest))
	mac.Write([]byte(uri))
	return hex.EncodeToString(mac.Sum(nil))
}

// GetUserAddress 获取用户充值地址
// currency:
// btc: BTC
// ltc: LTC
// eth: 以太币
// etc: ETC币
func (cb *CHBTC) GetUserAddress(currency string) (string, error) {
	url := "method=getUserAddress"
	url += "&accesskey=" + cb.AccessKey
	url += "&currency=" + currency
	sign := cb.Sign(url)
	url += "&sign=" + sign
	url += "&reqTime=" + strconv.FormatInt(time.Now().UnixNano()/(int64(time.Millisecond)/int64(time.Nanosecond)), 10)

	log.Debugf("Request url: %v", url)

	url = TradeAPI + "getUserAddress?" + url

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Debugf("Response body: %v", string(body))

	return gjson.GetBytes(body, "message.datas.key").String(), nil
}
