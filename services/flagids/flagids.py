#!/bin/env python
import os
from pathlib import Path

traffic_dir = Path(os.getenv("TULIP_TRAFFIC_DIR", "/traffic"))
tick_length = os.getenv("TICK_LENGTH", 2*60*1000)
start_date = os.getenv("TICK_START", "2018-06-27T13:00+02:00")
mongo_host = os.getenv("TULIP_MONGO", "localhost:27017")
mongo_server = f'mongodb://{mongo_host}/'
vm_ip = os.getenv("VM_IP", "10.10.3.1")
