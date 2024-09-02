#!/usr/bin/env python
# -*- coding: utf-8 -*-

import abc
import dataclasses
import json
import uuid
from datetime import datetime, timedelta
from ipaddress import IPv4Address, IPv6Address
from typing import Any


def encode(obj: Any) -> Any:
    if dataclasses.is_dataclass(obj):
        fields = dataclasses.fields(obj)
        return {f.name: getattr(obj, f.name) for f in fields}
    if isinstance(obj, timedelta):
        return obj.total_seconds()
    if isinstance(obj, datetime):
        return obj.isoformat()
    if isinstance(obj, uuid.UUID):
        return str(obj)
    if isinstance(obj, IPv4Address) or isinstance(obj, IPv6Address):
        return str(obj)
    return json.JSONEncoder().default(obj)


class JsonFactory(abc.ABC):
    @abc.abstractmethod
    def to_json(self) -> Any:
        return encode(self)


class Encoder(json.JSONEncoder):
    def default(self, o: Any) -> Any:
        if isinstance(o, JsonFactory):
            return o.to_json()
        return encode(o)


def dumps(*args, **kwargs) -> str:
    return json.dumps(*args, cls=Encoder, **kwargs)


def loads(*args, **kwargs) -> Any:
    return json.loads(*args, **kwargs)
