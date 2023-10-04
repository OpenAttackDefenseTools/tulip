#!/usr/bin/env python
# -*- coding: utf-8 -*-
from __future__ import annotations

import base64
import re
import uuid
from contextlib import contextmanager
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from ipaddress import IPv4Address, IPv6Address
from typing import Any, Iterator, cast

import dateutil.parser
import psycopg
import psycopg_pool
from psycopg import sql
from psycopg.rows import class_row, dict_row

import configurations
from json_util import JsonFactory


@dataclass(slots=True, kw_only=True)
class FlowQuery:
    regex_insensitive: re.Pattern | None = None
    ip_src: str | None = None
    ip_dst: str | None = None
    port_src: int | None = None
    port_dst: int | None = None
    time_from: datetime | None = None
    time_to: datetime | None = None
    tags_include: list[str] = field(default_factory=list)
    tags_exclude: list[str] = field(default_factory=list)


@dataclass(slots=True, kw_only=True)
class Flow:
    id: uuid.UUID
    time: datetime
    port_src: int
    port_dst: int
    ip_src: IPv4Address | IPv6Address
    ip_dst: IPv4Address | IPv6Address
    duration: timedelta
    blocked: bool
    pcap_id: uuid.UUID
    pcap_name: str
    link_parent_id: uuid.UUID
    link_child_id: uuid.UUID
    fingerprints: list[int]
    packets_count: int
    packets_size: int
    flags_in: int
    flags_out: int
    signatures: list[Signature]
    tags: list[str]


@dataclass(slots=True, kw_only=True)
class Signature:
    id: int
    message: str
    action: str


@dataclass(slots=True, kw_only=True)
class FlowItem(JsonFactory):
    id: uuid.UUID
    flow_id: uuid.UUID
    kind: str
    time: datetime
    direction: str
    data: bytes
    text: str

    def to_json(self) -> Any:
        result = JsonFactory.to_json(self)
        result["data"] = base64.b64encode(result["data"]).decode("ascii")
        return result


@dataclass(slots=True, kw_only=True)
class FlowDetail(Flow):
    items: list[FlowItem] = field(default_factory=list)

    def kind_items(self, kind: str = "raw") -> list[FlowItem]:
        return [i for i in self.items if i.kind == kind]

    def item_data(self, kind: str = "raw") -> list[bytes]:
        return [i.data for i in self.kind_items(kind)]

    def collect_data(self, kind: str = "raw") -> bytes:
        return b"".join(self.item_data(kind))


@dataclass(slots=True, kw_only=True)
class StatsQuery:
    service: str | None = None
    tick_from: int | None = None
    tick_to: int | None = None


@dataclass(slots=True)
class Stats:
    tick: int
    flow_count: int = 0
    tag_flag_in: int = 0
    tag_flag_out: int = 0
    tag_blocked: int = 0
    tag_suricata: int = 0
    tag_enemy: int = 0
    flag_in: int = 0
    flag_out: int = 0


class Pool(psycopg_pool.ConnectionPool):
    def __init__(self, connection_string: str, *, open: bool = False, **kwargs) -> None:
        super().__init__(
            connection_string,
            open=open,
            connection_class=Connection,
            **kwargs,
        )

    @contextmanager
    def connection(self, timeout: float | None = None) -> Iterator[Connection]:
        with super().connection(timeout) as connection:
            yield cast(Connection, connection)


class Connection(psycopg.Connection):
    def flow_query(self, query: FlowQuery) -> list[Flow]:
        conditions = [sql.SQL("true")]
        parameters = {}

        if query.regex_insensitive:
            parameters["regex_insensitive"] = query.regex_insensitive.pattern
            condition = """
                SELECT * FROM flow_item AS fi
                WHERE fi.flow_id = f.id
                    AND fi.text ~* %(regex_insensitive)s
            """
            conditions.append(sql.SQL(f"EXISTS ({condition})"))

        if query.ip_src:
            parameters["ip_src"] = query.ip_src  # TODO: Covert this to cidr match
            conditions.append(sql.SQL("f.ip_src = %(ip_src)s"))
        if query.ip_dst:
            parameters["ip_dst"] = query.ip_dst  # TODO: Covert this to cidr match
            conditions.append(sql.SQL("f.ip_dst = %(ip_dst)s"))

        if query.port_src:
            parameters["port_src"] = query.port_src
            conditions.append(sql.SQL("f.port_src = %(port_src)s"))
        if query.port_dst:
            parameters["port_dst"] = query.port_dst
            conditions.append(sql.SQL("f.port_dst = %(port_dst)s"))

        if query.time_from:
            parameters["time_from"] = query.time_from
            conditions.append(sql.SQL("f.id > uuid_pack_low(%(time_from)s)"))
        if query.time_to:
            parameters["time_to"] = query.time_to
            conditions.append(sql.SQL("f.id < uuid_pack_high(%(time_to)s)"))

        if query.tags_include:
            parameters["tags_include"] = query.tags_include
            conditions.append(sql.SQL("f.tags ?| %(tags_include)s"))
        if query.tags_exclude:
            parameters["tags_exclude"] = query.tags_exclude
            conditions.append(sql.SQL("NOT f.tags ?| %(tags_exclude)s"))

        # TODO: Indexes
        sql_query = sql.SQL(
            """
            SELECT f.*, p.name AS pcap_name
            FROM flow AS f
            INNER JOIN pcap AS p
                ON p.id = f.pcap_id
            WHERE {conditions}
            ORDER BY id DESC
            LIMIT 2000
        """
        ).format(conditions=sql.SQL(" AND ").join(conditions))
        with self.cursor(row_factory=class_row(Flow)) as cursor:
            flows = cursor.execute(sql_query, parameters).fetchall()

        # Filter out non-existing tags and sort the rest
        tags = self.tag_list()
        for flow in flows:
            flow.tags = list(filter(lambda t: t in flow.tags, tags))

        return flows

    def flow_detail(self, id: uuid.UUID) -> FlowDetail | None:
        # TODO: Indexes
        sql_query = """
            SELECT f.*, p.name AS pcap_name
            FROM flow AS f
            INNER JOIN pcap AS p
                ON p.id = f.pcap_id
            WHERE f.id = %(id)s
            ORDER BY id DESC
            LIMIT 2000
        """
        with self.cursor(row_factory=class_row(FlowDetail)) as cursor:
            flow = cursor.execute(sql_query, {"id": id}).fetchone()

        if flow is None:
            return None

        flow.items = self.flow_item_query(flow)

        # Filter out non-existing tags and sort the rest
        flow.tags = list(filter(lambda t: t in flow.tags, self.tag_list()))

        return flow

    def flow_item_query(self, flow: Flow) -> list[FlowItem]:
        # TODO: Indexes
        sql_query = """
            SELECT fi.*
            FROM flow_item AS fi
            WHERE fi.flow_id = %(flow_id)s
                AND fi.id > uuid_pack_low(%(time_start)s)
                AND fi.id < uuid_pack_high(%(time_end)s)
        """

        parameters = {
            "flow_id": flow.id,
            "time_start": flow.time,
            "time_end": flow.time + flow.duration,
        }

        with self.cursor(row_factory=class_row(FlowItem)) as cursor:
            return cursor.execute(sql_query, parameters).fetchall()

    def flow_tag(self, flow_id: uuid.UUID, tag: str, apply: bool) -> None:
        if apply:
            sql_query = """
                UPDATE flow
                SET tags = jsonb_unique(tags || jsonb_build_array(%(tag)s::text))
                WHERE id = %(flow_id)s
            """
        else:
            sql_query = """
                UPDATE flow
                SET tags = tags - %(tag)s::text
                WHERE id = %(flow_id)s
            """

        self.execute(sql_query, {"flow_id": flow_id, "tag": tag})

    def stats_query(self, query: StatsQuery) -> dict[int, Stats]:
        now = datetime.now(tz=timezone.utc)
        tick_first = dateutil.parser.parse(configurations.start_date)
        tick_length = timedelta(milliseconds=int(configurations.tick_length))
        tick_current = ((now - tick_first) // tick_length) + 1
        tick_start = query.tick_from if query.tick_from else 0
        tick_end = query.tick_to if query.tick_to else tick_current
        time_start = tick_first + (tick_start * tick_length)
        time_end = tick_first + (tick_end * tick_length)

        stats: dict[int, Stats] = {i: Stats(i) for i in range(tick_start, tick_end)}

        parameters = {
            "tick_length": tick_length,
            "tick_first": tick_first,
            "time_start": time_start,
            "time_end": time_end,
        }

        # TODO: Make an intelligent db function for tick ranges
        # TODO: Indexes
        sql_query = """
            SELECT (extract(epoch from (time_bucket(%(tick_length)s, time, origin => %(tick_first)s) - %(tick_first)s)) / extract(epoch from %(tick_length)s))::int AS tick,
                count(id) AS count, sum(flags_in) AS flags_in, sum(flags_out) AS flags_out
            FROM flow AS f
            WHERE f.id > uuid_pack_low(%(time_start)s)
                AND f.id < uuid_pack_high(%(time_end)s)
            GROUP BY tick
        """
        with self.cursor(row_factory=dict_row) as cursor:
            for row in cursor.execute(sql_query, parameters):
                stats[row["tick"]].flow_count = row["count"]
                stats[row["tick"]].flag_in = row["flags_in"]
                stats[row["tick"]].flag_out = row["flags_out"]

        # TODO: Maybe count all tags? The query already selects the numbers
        # TODO: Make an intelligent db function for tick ranges
        # TODO: Indexes
        sql_query = """
            SELECT time_bucket(%(tick_length)s, time, origin => %(tick_first)s) AS tick_start,
                (extract(epoch from (time_bucket(%(tick_length)s, time, origin => %(tick_first)s) - %(tick_first)s)) / extract(epoch from %(tick_length)s))::int AS tick,
                t.name AS tag, count(f.id) AS count
            FROM flow AS f
            JOIN tag AS t
                ON f.tags ? t.name
            WHERE f.id > uuid_pack_low(%(time_start)s)
                AND f.id < uuid_pack_high(%(time_end)s)
            GROUP BY tick_start, t.name
            ORDER BY tick ASC
        """
        with self.cursor(row_factory=dict_row) as cursor:
            for row in cursor.execute(sql_query, parameters):
                if row["tag"] == "flag-in":
                    stats[row["tick"]].tag_flag_in += row["count"]
                elif row["tag"] == "flag-out":
                    stats[row["tick"]].tag_flag_out += row["count"]
                elif row["tag"] == "blocked":
                    stats[row["tick"]].tag_blocked += row["count"]
                elif row["tag"] == "suricata":
                    stats[row["tick"]].tag_suricata += row["count"]
                elif row["tag"] == "enemy":
                    stats[row["tick"]].tag_enemy += row["count"]

        return stats

    def tag_list(self) -> list[str]:
        with self.cursor(row_factory=dict_row) as cursor:
            tags = cursor.execute("SELECT name FROM tag ORDER BY sort ASC").fetchall()
            return [t["name"] for t in tags]
