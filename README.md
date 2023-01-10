# filplus
> 不支持查询 glif 免费领取的 datacap 额度  
> 默认添加生产5个节点  
> 支持自动将 f3/f1 地址查出 ID，查询接口使用 [Glif](https://api.node.glif.io/rpc/v0)

## 编译
```bash
make

# 编译 amd64 平台
make linux
```

## 查询使用
```bash
./filplus query --sp f01877571,f01880047 --client f01860990,f01817237

client            sp                datacap(T)
f01860990         f01877571         258.40625
f01860990         f01880047         314.6875
f01817237         f01877571         56.71875
f01817237         f01880047         56.25
Total Datacap:    686.0625


./filplus query --client f01860990,f01817237

client            sp                datacap(T)
f01860990         f01877571         258.40625
f01860990         f01878005         315.3125
f01860990         f01880047         314.6875
f01860990         f01882177         315.3125
f01860990         f01882184         314.71875
f01817237         f01877571         56.71875
f01817237         f01878005         56.25
f01817237         f01880047         56.25
f01817237         f01882177         56.25
f01817237         f01882184         56.3125
Total Datacap:    1800.21875


# 使用参数简写查询额度
./filplus query -c f3w4wlayytfmsay6gu5phhij5r4yyx7t4xxrlosgotlmqg5eih3co5atsra4h2pe4qd2d6c76bvhj6nwim7lgq -s f01132084,f01169691

client            sp                datacap(T)
f01358347         f01132084         0.0146484375
f01358347         f01169691         0.72314453125
Total Datacap:    0.73779296875


# 使用 --lookup=false 参数原样输出输入的地址
$ ./filplus query -c f3w4wlayytfmsay6gu5phhij5r4yyx7t4xxrlosgotlmqg5eih3co5atsra4h2pe4qd2d6c76bvhj6nwim7lgq -s f01132084,f01169691  -l=false

client                                                                                    sp                datacap(T)
f3w4wlayytfmsay6gu5phhij5r4yyx7t4xxrlosgotlmqg5eih3co5atsra4h2pe4qd2d6c76bvhj6nwim7lgq    f01132084         0.0146484375
f3w4wlayytfmsay6gu5phhij5r4yyx7t4xxrlosgotlmqg5eih3co5atsra4h2pe4qd2d6c76bvhj6nwim7lgq    f01169691         0.72314453125
Total Datacap:  
                                                                          0.73779296875
```
## 计算使用
**从lotus导出deal文件**
```bash
curl --location --request POST 'http://10.126.1.15:1234/rpc/v0' --header 'Content-Type: application/json' --data-raw '{
     "jsonrpc": "2.0",
     "method": "Filecoin.StateMarketDeals",
     "params": [
     [
     ]
   ],
     "id": 1
   }' >deal.list
```
```bash
$ ./filplus calculate --file ~/Downloads/deal.list -s t01000
2020-08-25 06:00:00 ~ 2060-08-25 06:00:00
client            sp                datacap(T)
t0100             t01000            24
Total Datacap                       24

$ ./filplus calculate --file ~/Downloads/deal.list -s t01000,t01001
2020-08-25 06:00:00 ~ 2060-08-25 06:00:00
client            sp                datacap(T)
t0100             t01000            24
t0101             t01001            24
Total Datacap                       48

$ ./filplus calculate --file ~/Downloads/deal.list -s t01000,t01001 -c t0100
2020-08-25 06:00:00 ~ 2060-08-25 06:00:00
client            sp                datacap(T)
t0100             t01000            24
Total Datacap                       24

$ ./filplus calculate --file ~/Downloads/deal.list -s t01000,t01001 -c t0100,t0101
2020-08-25 06:00:00 ~ 2060-08-25 06:00:00
client            sp                datacap(T)
t0100             t01000            24
t0101             t01001            24
Total Datacap                       48
```