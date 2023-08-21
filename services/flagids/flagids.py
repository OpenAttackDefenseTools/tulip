#!/bin/env python
import os
import re
import time
import pymongo
import requests
from datetime import datetime

DELAY = 5 # DELAY from start of tick
tick_length = int(os.getenv("TICK_LENGTH", 10*1000))//1000
start_date = os.getenv("TICK_START", "2018-06-27T13:00+02:00")
mongo_host = os.getenv("TULIP_MONGO", "localhost:27017").split(':')
vm_ip = os.getenv("VM_IP", "10.10.3.1")
flagid_endpoint = os.getenv("FLAGID_ENDPOINT", "http://localhost:8000/flagids.json")

print('STARTING FLAGIDS')
client = pymongo.MongoClient(mongo_host[0], int(mongo_host[1]))
db = client['pcap']
print('CONNECTION TO MONGO ESTABLISHED')

IP_PATTERN = re.compile(r'^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$')

# check if a string is an valid ip
def is_ip(ip):
    return IP_PATTERN.match(ip) is not None

# get leaf nodes of a json data struct
def get_leaf_nodes(data):
    if isinstance(data, dict):
        for key, value in data.items():
            if is_ip(key) and key != vm_ip:
                continue
            yield from get_leaf_nodes(value)
    elif isinstance(data, list):
        for item in data:
            yield from get_leaf_nodes(item)
    else:
        # prevent ips from being used as Flagids
        if not is_ip(data):
            yield data

def update_flagids():
    print('Updating flagids: ', time.time())
    response = requests.get(flagid_endpoint)
    crnt_time = int(time.time())
    nodes = [{"_id": node, "time": crnt_time} for node in  get_leaf_nodes(response.json())]
    db['flagids'].insert_many(nodes)

def main():
    start_datetime = datetime.strptime(start_date, '%Y-%m-%dT%H:%M%z')
    unixtime = time.mktime(start_datetime.timetuple())
    while True:
        try:
            update_flagids()
            crnt_time = time.time()
            time_diff = crnt_time - unixtime
            wait = DELAY + tick_length * (time_diff//tick_length) + time_diff%tick_length
            time.sleep(wait)
        except Exception as e:
            print('ERROR: ', e)
            time.sleep(10)

if __name__ == '__main__':
    main()