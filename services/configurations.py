import re
import os

ws_ip = os.getenv("REACT_APP_FLOWER_SERVER_IP", "0.0.0.0")
mongo_server = 'mongodb://' + ws_ip + ':27017/'
vm_ip = "10.10.3.1"  # todo put regex

services = [{"ip": vm_ip, "port": 9876, "name": "cc_market"},
            {"ip": vm_ip, "port": 80, "name": "maze"},
            {"ip": vm_ip, "port": 8080, "name": "scadent"},
            {"ip": vm_ip, "port": 5000, "name": "starchaser"},
            {"ip": vm_ip, "port": 1883, "name": "scadnet_bin"}]


def containsFlag(text):
    # todo implementare logica contains
    regex_flag = os.getenv("REACT_APP_FLAG_REGEX", r'[A-Z0-9]{31}=')
    match = re.match(regex_flag, text)
    return match
