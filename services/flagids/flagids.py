#!/bin/env python
import os
import time
from datetime import datetime

import psycopg_pool
import requests

DELAY = 5  # DELAY from start of tick
tick_length = int(os.getenv("TICK_LENGTH", 10 * 1000)) // 1000
start_date = os.getenv("TICK_START", "2018-06-27T13:00+02:00")
team_id = os.getenv("TEAM_ID", "10.10.3.1")
team_id_is_digit = team_id.isdigit()
team_id_int = int(team_id) if team_id_is_digit else None
flagid_endpoint = os.getenv("FLAGID_ENDPOINT", "http://localhost:8000/flagids.json")
flagid_scrape_enabled = os.getenv("FLAGID_SCRAPE", "") != ""

client = None
db = None
if flagid_scrape_enabled:
    print("STARTING FLAGIDS")
    print("CONFIG:")
    print("  DELAY: ", DELAY)
    print("  TICK_LENGTH: ", tick_length)
    print("  TICK_START: ", start_date)
    print("  TIMESCALE: ", os.environ.get("TIMESCALE"))
    print("  TEAM_ID: ", team_id)
    print("  FLAGID_ENDPOINT: ", flagid_endpoint)
    db = psycopg_pool.ConnectionPool(os.environ["TIMESCALE"])
    print("CONNECTION TO MONGO ESTABLISHED", flush=True)
else:
    print("FLAGID SCRAPE DISABLED", flush=True)


# get leaf nodes of a json data struct
def get_leaf_nodes(data):
    if isinstance(data, dict):
        if team_id in data.keys():
            yield from get_leaf_nodes(data[team_id])
        elif team_id_is_digit and team_id_int in data.keys():
            yield from get_leaf_nodes(data[team_id_int])
        else:
            for value in data.values():
                yield from get_leaf_nodes(value)
    elif isinstance(data, list):
        if team_id in data or (team_id_is_digit and team_id_int in data):
            yield
        else:
            for item in data:
                print(item, end=" ", flush=True)
                yield from get_leaf_nodes(item)
    else:
        # prevent id from being used as Flagids
        yield data


def update_flagids():
    assert db is not None

    response = requests.get(flagid_endpoint)
    rows = [(node,) for node in get_leaf_nodes(response.json()) if node is not None]
    print("Updating flagids: ", time.time(), f"({len(rows)})", flush=True)

    with db.connection() as conn:
        with conn.cursor() as cur:
            cur.executemany("INSERT INTO flag_id (content) VALUES (%s)", rows)
            conn.commit()


def main():
    start_datetime = datetime.strptime(start_date, "%Y-%m-%dT%H:%M:%S%z")
    unixtime = time.mktime(start_datetime.timetuple())
    while True:
        try:
            if flagid_scrape_enabled:
                update_flagids()
            crnt_time = time.time()
            time_diff = max(0, crnt_time - unixtime)
            wait = (
                DELAY
                + tick_length * (time_diff // tick_length)
                + time_diff % tick_length
            )
            time.sleep(wait)
        except Exception as e:
            print("ERROR: ", e, flush=True)
            time.sleep(10)


if __name__ == "__main__":
    main()
