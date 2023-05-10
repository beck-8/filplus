import json
import requests
import time
import sys
from concurrent.futures import ThreadPoolExecutor

# 需要用修改过的Lotus，因为有个接口要用到
url = "http://128.136.157.164:51234/rpc/v0"
# url = 'https://api.node.glif.io/rpc/v0'
# filecoin主网0区块时间
bootstrap_time = time.mktime(time.strptime(
    '2020-8-25 06:00:00', '%Y-%m-%d %H:%M:%S'))


def StateMinerActiveSectors(mid):
    # Expiration 参数并不一定准确
    payload = json.dumps({
        "jsonrpc": "2.0",
        "method": "Filecoin.StateMinerActiveSectors",
        "params": [
            mid,
            []
        ],
        "id": 1
    })
    headers = {
        'Content-Type': 'application/json'
    }

    response = requests.request("POST", url, headers=headers, data=payload)

    if response.status_code == 200:
        return response.json()['result']
    else:
        print(response.status_code, response.text)


def StateMinerSectors(mid):
    payload = json.dumps({
        "jsonrpc": "2.0",
        "method": "Filecoin.StateMinerSectors",
        "params": [
            mid,
            None,
            []
        ],
        "id": 1
    })
    headers = {
        'Content-Type': 'application/json'
    }

    response = requests.request("POST", url, headers=headers, data=payload)

    if response.status_code == 200:
        return response.json()['result']
    else:
        print(response.status_code, response.text)


def StateMinerInfo(mid):
    payload = json.dumps({
        "jsonrpc": "2.0",
        "method": "Filecoin.StateMinerInfo",
        "params": [
            mid,
            []
        ],
        "id": 1
    })
    headers = {
        'Content-Type': 'application/json'
    }

    response = requests.request("POST", url, headers=headers, data=payload)

    if response.status_code == 200:
        return response.json()['result']
    else:
        print(response.status_code, response.text)


def StateMinerPartitionsUint(mid, deadline):
    payload = json.dumps({
        "jsonrpc": "2.0",
        "method": "Filecoin.StateMinerPartitionsUint",
        "params": [
            mid,
            deadline,
            []
        ],
        "id": 1
    })
    headers = {
        'Content-Type': 'application/json'
    }

    response = requests.request("POST", url, headers=headers, data=payload)

    if response.status_code == 200:
        return response.json()['result']
    else:
        print(response.status_code, response.text)


def height_to_time(height):
    # 输入 高度
    timestamp = bootstrap_time + height * 30
    time_local = time.localtime(timestamp)
    dt = time.strftime('%Y-%m-%d', time_local)
    return dt


def time_to_height(num=0):
    # 输入类似格式时间 '2020-8-25 06:00:00'
    cur_time = time.time()
    timestamp = cur_time - cur_time % 86400 - 3600*8 - bootstrap_time
    start_epoch = int(timestamp / 30)
    end_epoch = start_epoch + 2880 * num
    return start_epoch, end_epoch


# 计算哪些扇区质押币高，需配合compute num != 0 使用（即使用StateMinerActiveSectors）
def compute_sectors(num, expire_dict):
    sectors_dict = {}
    for dt in expire_dict:
        for id in expire_dict[dt]:
            sectors_dict[id] = expire_dict[dt][id]
    result_list = sorted(sectors_dict.items(), key=lambda kv: (kv[1], kv[0]))
    if num > len(result_list):
        num = len(result_list)
    for i in range(num-1, -1, -1):
        sys.stdout.write(f'{result_list[i]}\n')


def compute(mid, num=0):
    expire_dict = {}
    miner_info = StateMinerInfo(mid)
    SectorSize = miner_info['SectorSize'] / 1024 ** 3
    cur_epoch,_ = time_to_height()

    # 取扇区对应的deadline
    deadlines = {}
    if num != 0:
        for dead in range(48):
            deadline = StateMinerPartitionsUint(mid, dead)
            if deadline != None:
                for part in deadline:
                    for id in part['LiveSectors']:
                        deadlines[id] = dead

        for i in StateMinerActiveSectors(mid):
            SectorNumber = i['SectorNumber']
            SealProof = i['SealProof']
            Activation = i['Activation']
            Expiration = i['Expiration'] + 60*deadlines[SectorNumber]
            InitialPledge = int(i['InitialPledge'])
            ExpectedDayReward = int(i['ExpectedDayReward'])
            ExpectedStoragePledge = int(i['ExpectedStoragePledge'])
            secotr_time = height_to_time(Expiration)

            start_epoch, end_epoch = time_to_height(num)
            if start_epoch <= Expiration <= end_epoch:
                if secotr_time not in expire_dict:
                    expire_dict[secotr_time] = {}

                    # 初始化惩罚数值
                    expire_dict[secotr_time]["penalty"] = 0
                expire_dict[secotr_time][SectorNumber] = InitialPledge

                
                live_dur = (cur_epoch-Activation)/2880
                if live_dur >= 140 :
                    expire_dict[secotr_time]["penalty"] += ExpectedStoragePledge + 70 * ExpectedDayReward
                else:
                    expire_dict[secotr_time]["penalty"] += ExpectedStoragePledge + live_dur/2 * ExpectedDayReward


        # 计算质押币前多少个扇区
        # compute_sectors(5,expire_dict)
    else:
        for dead in range(48):
            deadline = StateMinerPartitionsUint(mid, dead)
            if deadline != None:
                for part in deadline:
                    for id in part['AllSectors']:
                        deadlines[id] = dead

        for i in StateMinerSectors(mid):
            SectorNumber = i['SectorNumber']
            SealProof = i['SealProof']
            Activation = i['Activation']
            Expiration = i['Expiration'] + 60*deadlines[SectorNumber]
            InitialPledge = int(i['InitialPledge'])
            ExpectedDayReward = int(i['ExpectedDayReward'])
            ExpectedStoragePledge = int(i['ExpectedStoragePledge'])
            secotr_time = height_to_time(Expiration)

            if secotr_time not in expire_dict:
                expire_dict[secotr_time] = {}

                # 初始化惩罚数值
                expire_dict[secotr_time]["penalty"] = 0
            expire_dict[secotr_time][SectorNumber] = InitialPledge

            live_dur = (cur_epoch-Activation)/2880
            if live_dur >= 140 :
                expire_dict[secotr_time]["penalty"] += ExpectedStoragePledge + 70 * ExpectedDayReward
            else:
                expire_dict[secotr_time]["penalty"] += ExpectedStoragePledge + live_dur/2 * ExpectedDayReward
            

    for date in expire_dict:

        sectors_sum = len(expire_dict[date])
        penalty = expire_dict[date]["penalty"]
        fil_sum = 0
        for i in expire_dict[date]:
            if i != "penalty":
                fil_sum += expire_dict[date][i]
        sys.stdout.write("{:<12}{:<12}{:<8}{:<16.4f}{:<24}{:<6}\n".format(
            date, mid, sectors_sum, sectors_sum*SectorSize/1024, fil_sum/1e18,penalty/1e18))
        # print("{:<12}{:<12}{:<8}{:<10.4f}{:<6}".format(date,mid, sectors_sum,sectors_sum*SectorSize/1024, fil_sum/1e18))


# compute(xx) 计算从当日到 XX 日，每日扇区过期量、质押释放量
# compute(0) 计算节点所有日期的扇区过期量、质押释放量（包括已经过期的）
if __name__ == '__main__':
    pool = ThreadPoolExecutor(max_workers=10)
    owner_list = [
                'f010038',
                  'f010202',
                  'f014686',
                  'f014699',
                  'f029585',
                  'f01155',
                  'f082730',
                  'f033462',
                  'f0145060',
                  'f060805',
                  'f086204',
                  'f0134867',
                  'f0130639',
                  'f0685539',
                  'f01086808',
                  'f01135819',
                  'f01128375',
                  'f01149873',
                  'f01201224',
                  'f01478558',
                  'f01482593',
                  'f01594217',
                  'f01674133',
                  'f01688066',
                  'f01756175',
                  'f01784928',
                  'f01784929',
                  'f01784930',
                  'f01813052',
                  "f01877571", "f01878005", "f01880047","f01882177","f01882184",

                  ]
    # expire_dict = {'2022-4-21':{0:1792465} }

    for i in owner_list:
        pool.submit(compute, i, 0)
        # compute(i,0)
