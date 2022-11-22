# sounding - a cryptocurrency exchange listener in Go
[![build](https://github.com/oerlikon/sounding/actions/workflows/ci.yml/badge.svg)](https://github.com/oerlikon/sounding/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/oerlikon/sounding)](https://goreportcard.com/report/github.com/oerlikon/sounding)
[![License: Unlicense](https://img.shields.io/badge/license-Unlicense-blue.svg)](http://unlicense.org/)

This program connects to Binance, Bitfinex and Kraken public WebSocket APIs and listens to book and trade updates for specified instruments, printing them in unified format to stdout in the order they arrive. Output is very buffered and gets flushed when the program is interrupted.

Symbol names are what each exchange's API expects to identify their instruments.

Run it like:
```
./sound binance:btcusdt bitfinex:btcusd kraken:xbt/usd > _
```
or like:
```
echo binance:btcusdt bitfinex:btcusd kraken:xbt/usd | ./sound > _
```

To get something like:
```
B 1668980335932,2022-11-20T21:38:55.932Z,Binance,BTCUSDT,BID,16442.15000000,0.01571000
B 1668980349216,2022-11-20T21:39:09.216Z,Binance,BTCUSDT,ASK,16454.38000000,0.00000000
B 1668980351333,2022-11-20T21:39:11.333Z,Bitfinex,BTCUSD,BID,16442,0.1427021
B 1668980357196,2022-11-20T21:39:17.196Z,Bitfinex,BTCUSD,ASK,16450,6.10398366
B 1668980398219,2022-11-20T21:39:58.219Z,Binance,BTCUSDT,BID,16404.49000000,0.00800000
B 1668980409947,2022-11-20T21:40:09.947Z,Bitfinex,BTCUSD,BID,16381,0
B 1668980422219,2022-11-20T21:40:22.219Z,Binance,BTCUSDT,BID,16445.49000000,0.00000000
B 1668980425220,2022-11-20T21:40:25.220Z,Binance,BTCUSDT,BID,16450.82000000,0.01742000
B 1668980427381,2022-11-20T21:40:27.381Z,Bitfinex,BTCUSD,ASK,16447,2.90763644
B 1668980429691,2022-11-20T21:40:29.691Z,Bitfinex,BTCUSD,BID,16429,4.69797936
B 1668980433219,2022-11-20T21:40:33.219Z,Binance,BTCUSDT,ASK,16451.87000000,0.00600000
B 1668980437220,2022-11-20T21:40:37.220Z,Binance,BTCUSDT,BID,16442.95000000,0.00689000
B 1668980442220,2022-11-20T21:40:42.220Z,Binance,BTCUSDT,ASK,16449.38000000,0.01153000
B 1668980444220,2022-11-20T21:40:44.220Z,Binance,BTCUSDT,ASK,16450.81000000,0.00000000
B 1668980444698,2022-11-20T21:40:44.698Z,Bitfinex,BTCUSD,ASK,16466,0.22746226
B 1668980452220,2022-11-20T21:40:52.220Z,Binance,BTCUSDT,ASK,16449.70000000,0.08067000
T 1668980460024,2022-11-20T21:41:00.024Z,Binance,BTCUSDT,2216251376,15712375905,15712376072,SELL,16447.98000000,0.00354000
B 1668980460221,2022-11-20T21:41:00.221Z,Binance,BTCUSDT,BID,16445.78000000,0.00000000
T 1668980462423,2022-11-20T21:41:02.423Z,Binance,BTCUSDT,2216251667,15712377308,15712377519,SELL,16451.01000000,0.00581000
T 1668980464164,2022-11-20T21:41:04.164Z,Binance,BTCUSDT,2216251855,15712378596,15712378612,SELL,16448.62000000,0.00069000
B 1668980466222,2022-11-20T21:41:06.222Z,Binance,BTCUSDT,ASK,16452.06000000,0.00000000
B 1668980467717,2022-11-20T21:41:07.717Z,Kraken,XBT/USD,ASK,16534.40000,0.89000000
B 1668980473221,2022-11-20T21:41:13.221Z,Binance,BTCUSDT,BID,16443.47000000,0.30452000
B 1668980478222,2022-11-20T21:41:18.222Z,Binance,BTCUSDT,ASK,16446.82000000,0.00000000
B 1668980483127,2022-11-20T21:41:23.127Z,Kraken,XBT/USD,ASK,16426.60000,0.50000000
B 1668980488781,2022-11-20T21:41:28.781Z,Kraken,XBT/USD,ASK,16432.80000,1.51524830
B 1668980490223,2022-11-20T21:41:30.223Z,Binance,BTCUSDT,ASK,16455.22000000,1.02716000
B 1668980490523,2022-11-20T21:41:30.523Z,Bitfinex,BTCUSD,ASK,16661,0.20425275
B 1668980498223,2022-11-20T21:41:38.223Z,Binance,BTCUSDT,BID,16440.90000000,0.04560000
B 1668980506870,2022-11-20T21:41:46.870Z,Bitfinex,BTCUSD,ASK,16454,0.3330324
T 1668980513592,2022-11-20T21:41:53.592Z,Binance,BTCUSDT,2216254064,15712394038,15712394027,BUY,16454.25000000,0.00499000
B 1668980521425,2022-11-20T21:42:01.425Z,Kraken,XBT/USD,BID,16430.00000,0.13855717
B 1668980523217,2022-11-20T21:42:03.217Z,Kraken,XBT/USD,BID,16339.40000,15.30037937
B 1668980523224,2022-11-20T21:42:03.224Z,Binance,BTCUSDT,BID,16451.68000000,0.01269000
```
