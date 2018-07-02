#!/usr/bin/env python2
#Script that import pcap into flower dg
import json
import nids #good luck installing pynids!
import sys
import string
import pprint
import time

from configurations import containsFlag
from db import DB

end_states = (nids.NIDS_CLOSE, nids.NIDS_TIMEOUT, nids.NIDS_RESET)
db = DB()

data_flow = {}
filename = ""
flows_to_import = []
ts = {}
contains_flag = {}
done = 0
start_time = {}
inx = 0


def handleTcpStream(tcp):
    global data_flow, ts, contains_flag, done, start_time, inx

    if tcp.nids_state == nids.NIDS_JUST_EST:
        tcp.client.collect = 1
        tcp.server.collect = 1
        data_flow[tcp.addr] = []
        start_time[tcp.addr] = int(float(nids.get_pkt_ts()) * 1000)
        contains_flag[tcp.addr] = False
    elif tcp.nids_state == nids.NIDS_DATA:
        actor = tcp.client if tcp.client.count_new > 0 else tcp.server

        cnt = actor.count_new
        data = actor.data[:cnt]
        printable_data = ''.join([i if i in string.printable else '\\x{:02x}'.format(ord(i)) for i in data])
        name = "c" if actor is tcp.client else "s"

        last_flow = (data_flow[tcp.addr] or [None])[-1]
        #this is from server, and last one is from server. Just concatenate data
        if last_flow and last_flow["from"] == name: 
            data_flow[tcp.addr][-1]["data"] += printable_data
            data_flow[tcp.addr][-1]["hex"] += data.encode("hex")
        else:
            data_flow[tcp.addr].append(
                {"from": name,
                 "data": printable_data,
                 "hex": data.encode("hex"),
                 "time": int(float(nids.get_pkt_ts()) * 1000)
                 }
            )
        #only if this we don't know if this flow contains a flag
        if not contains_flag[tcp.addr] and containsFlag(data):
            contains_flag[tcp.addr] = True

        tcp.discard(actor.count_new)

    elif tcp.nids_state in end_states:
        ((src, sport), (dst, dport)) = tcp.addr

        done += 1
        if done % 100 == 0: print(done)
        if len(data_flow[tcp.addr]) == 0:
            return

        ts = int(float(nids.get_pkt_ts()) * 1000)

        flow = {"inx": inx,
                "filename": filename,
                "src_ip": src,
                "src_port": sport,
                "dst_ip": dst,
                "dst_port": dport,
                "time": start_time[tcp.addr],
                "duration": (ts - start_time[tcp.addr]),
                "contains_flag": contains_flag[tcp.addr],
                "starred": 0,
                "flow": data_flow[tcp.addr]
                }

        flows_to_import.append(flow)
        del data_flow[tcp.addr]
        #TODO check if each flow is less than 16 MB (mongodb document limit)


nids.param("pcap_filter", "tcp")  # restrict to TCP only
nids.chksum_ctl([('0.0.0.0/0', False)])  # disable checksumming

if len(sys.argv) == 2:
    filename = sys.argv[1]
    if "./" in filename:
        filename = filename[2:]
    print("importing pcaps from " + filename)
    nids.param("filename", filename)
else:
    print("pcap file required")
    exit()

nids.init()
nids.register_tcp(handleTcpStream)
nids.run()

print("importing " + str(len(flows_to_import)) + " flows into mongodb!")
db.insertFlows(filename, flows_to_import)
db.setFileImported(filename)
