#!/usr/bin/env python
# -*- coding: utf-8 -*-
from __future__ import annotations

import base64
import re
import uuid
from contextlib import contextmanager
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from ipaddress import IPv4Address, IPv4Network, IPv6Address, IPv6Network
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
    ip_src: IPv4Network | IPv6Network | None = None
    ip_dst: IPv4Network | IPv6Network | None = None
    port_src: int | None = None
    port_dst: int | None = None
    time_from: datetime | None = None
    time_to: datetime | None = None
    tags_include: list[str] = field(default_factory=list)
    tags_exclude: list[str] = field(default_factory=list)
    tag_intersection_and: bool = False
    limit: int = 1000


@dataclass(slots=True, kw_only=True)
class Flow:
    id: uuid.UUID
    time: datetime
    port_src: int
    port_dst: int
    ip_src: IPv4Address | IPv6Address
    ip_dst: IPv4Address | IPv6Address
    duration: timedelta
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
    flags: list[str]
    flagids: list[str]
    rank: int = 0


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
        pre_select = sql.SQL(
            "WITH f AS (SELECT *, fid_rank_desc(id) AS rank FROM flow ORDER BY id DESC)"
        )
        conditions = [sql.SQL("true")]
        pre_conditions = [sql.SQL("true")]
        parameters = {}

        if query.ip_src:
            parameters["ip_src"] = query.ip_src
            conditions.append(sql.SQL("f.ip_src <<= %(ip_src)s"))
        if query.ip_dst:
            parameters["ip_dst"] = query.ip_dst
            conditions.append(sql.SQL("f.ip_dst <<= %(ip_dst)s"))

        if query.port_src:
            parameters["port_src"] = query.port_src
            conditions.append(sql.SQL("f.port_src = %(port_src)s"))
        if query.port_dst:
            parameters["port_dst"] = query.port_dst
            conditions.append(sql.SQL("f.port_dst = %(port_dst)s"))

        if query.time_from:
            parameters["time_from"] = query.time_from
            conditions.append(sql.SQL("f.id > fid_pack_low(%(time_from)s)"))
            pre_conditions.append(sql.SQL("flow_id > fid_pack_low(%(time_from)s)"))
        if query.time_to:
            parameters["time_to"] = query.time_to
            conditions.append(sql.SQL("f.id < fid_pack_high(%(time_to)s)"))
            pre_conditions.append(sql.SQL("flow_id < fid_pack_high(%(time_to)s)"))

        if query.tags_include:
            parameters["tags_include"] = query.tags_include
            if query.tag_intersection_and:
                conditions.append(sql.SQL("f.tags ?& %(tags_include)s"))
            else:
                conditions.append(sql.SQL("f.tags ?| %(tags_include)s"))
        if query.tags_exclude:
            parameters["tags_exclude"] = query.tags_exclude
            conditions.append(sql.SQL("NOT f.tags ?| %(tags_exclude)s"))

        if query.regex_insensitive:
            parameters["regex_insensitive"] = query.regex_insensitive.pattern
            text = """
                WITH fi AS (
                    SELECT flow_id, fid_rank_desc(flow_id) AS rank
                    FROM flow_index
                    WHERE text ~* %(regex_insensitive)s
                        AND {pre_conditions}
                    ORDER BY rank
                ), fd AS (
                    SELECT DISTINCT flow_id, rank
                    FROM fi
                ), f AS (
                    SELECT fl.*, fd.rank
                    FROM fd
                    LEFT JOIN flow AS fl
                        ON fl.id = fd.flow_id
                )
            """
            pre_select = sql.SQL(text).format(
                pre_conditions=sql.SQL(" AND ").join(pre_conditions)
            )

        text_query = """
            /*+
                IndexScan(flow_index)
                Set(enable_material false)
            */
            {pre_select}
            SELECT f.*, p.name AS pcap_name
            FROM f
            LEFT JOIN pcap AS p
                ON p.id = f.pcap_id
            WHERE {conditions}
            LIMIT {limit}
        """

        sql_query = sql.SQL(text_query).format(
            conditions=sql.SQL(" AND ").join(conditions),
            pre_select=pre_select,
            limit=query.limit,
        )

        with self.cursor(row_factory=class_row(Flow)) as cursor:
            flows = cursor.execute(sql_query, parameters).fetchall()

        # Filter out non-existing tags
        tags = self.tag_list()
        for flow in flows:
            flow.tags = list(filter(lambda t: t in flow.tags, tags))

        return list(sorted(flows, key=lambda f: f.rank))

    def flow_detail(self, id: uuid.UUID) -> FlowDetail | None:
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
        sql_query = """
            SELECT fi.*
            FROM flow_item AS fi
            WHERE fi.flow_id = %(flow_id)s
                AND fi.id > fid_pack_low(%(time_start)s)
                AND fi.id < fid_pack_high(%(time_end)s)
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

        sql_query = """
            SELECT tick_number_bucket(%(tick_first)s, %(tick_length)s, time) AS tick,
                count(id) AS count, sum(flags_in) AS flags_in, sum(flags_out) AS flags_out
            FROM flow AS f
            WHERE f.id > fid_pack_low(%(time_start)s)
                AND f.id < fid_pack_high(%(time_end)s)
            GROUP BY tick
        """
        with self.cursor(row_factory=dict_row) as cursor:
            for row in cursor.execute(sql_query, parameters):
                stats[row["tick"]].flow_count = row["count"]
                stats[row["tick"]].flag_in = row["flags_in"]
                stats[row["tick"]].flag_out = row["flags_out"]

        # TODO: Maybe count all tags? The query already selects the numbers
        sql_query = """
            SELECT tick_time_bucket(%(tick_first)s, %(tick_length)s, time) AS tick_start,
                tick_number_bucket(%(tick_first)s, %(tick_length)s, time) AS tick,
                t.name AS tag, count(f.id) AS count
            FROM flow AS f
            JOIN tag AS t
                ON f.tags ? t.name
            WHERE f.id > fid_pack_low(%(time_start)s)
                AND f.id < fid_pack_high(%(time_end)s)
            GROUP BY tick_start, tick, t.name
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
