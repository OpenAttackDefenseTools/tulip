#!/usr/bin/env python
# -*- coding: utf-8 -*-

# This file is part of Flower.
#
# Copyright ©2018 Nicolò Mazzucato
# Copyright ©2018 Antonio Groza
# Copyright ©2018 Brunello Simone
# Copyright ©2018 Alessio Marotta
# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS FILE HEADER.
#
# Flower is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Flower is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Flower.  If not, see <https://www.gnu.org/licenses/>.

from http.server import BaseHTTPRequestHandler
from urllib.parse import parse_qs
from jinja2 import Environment, BaseLoader
from io import BytesIO
import json

from database import FlowDetail

DISCARD_COOKIES = ["PHPSESSID", "wordpress_logged_in_", "session"]


HEADER_TEMPLATE = """import json
import os
import sys

import requests

HOST = os.getenv('TARGET_IP')
EXTRA = json.loads(os.getenv('TARGET_EXTRA', '[]'))
{% if use_requests_session %}
s = requests.Session()
{% endif -%}
"""

REQUEST_TEMPLATE = """
{{"s." if use_requests_session}}headers = {{headers}}
{% if data -%}
data = {{data}}
{% endif -%}
{{"res = " if print_info}}{{"s" if use_requests_session else "requests"}}.{{request_method}}(f"http://{HOST}:{{port}}" + {{request_path_repr}}{% if data %}, {{data_param_name}}=data{% endif %}{{ ", headers=headers" if not use_requests_session}})
{% if print_info -%}
print(res.text)
print(res.status_code, res.headers)
{% endif %}
"""


def render(template, **kwargs):
    return Environment(loader=BaseLoader()).from_string(template).render(kwargs)


# class to parse request informations
class HTTPRequest(BaseHTTPRequestHandler):
    def __init__(self, raw_http_request: bytes):
        self.rfile = BytesIO(raw_http_request)
        self.raw_requestline = self.rfile.readline()
        self.error_code = self.error_message = None
        self.parse_request()

        self.headers: dict[str, str]
        try:
            self.headers = dict(self.headers)
        except AttributeError:
            self.headers = {}

        # Data
        try:
            self.body = raw_http_request.split(b"\r\n\r\n", 1)[1].rstrip()
        except IndexError:
            self.body = None

    def send_error(self, code, message=None, explain=None):
        self.error_code = code
        self.error_message = message


def decode_http_request(raw_request: bytes, tokenize):
    request = HTTPRequest(raw_request)
    headers = {}
    blocked_headers = [
        "content-length",
        "accept-encoding",
        "connection",
        "accept",
        "host",
    ]
    content_type = ""
    data = None
    data_param_name = None

    for i in request.headers:
        normalized_header = i.lower()

        if normalized_header == "content-type":
            content_type = request.headers[i]
        if not normalized_header in blocked_headers:
            headers[i] = request.headers[i]

    # if tokenization is enabled and body is not empty, try to decode form body or JSON body
    if tokenize and request.body:
        # try to deserialize form data
        if content_type.startswith("application/x-www-form-urlencoded"):
            data_param_name = "data"
            data = {}
            body_dict = parse_qs(request.body.decode())
            for key, value in body_dict.items():
                if len(value) == 1:
                    data[key] = value[0]
                else:
                    data[key] = value

        # try to deserialize json
        if content_type.startswith("application/json"):
            data_param_name = "json"
            try:
                data = json.loads(request.body)
            except json.decoder.JSONDecodeError:
                pass

        # Forms with files are not yet implemented
        # # try to extract files
        # if content_type.startswith("multipart/form-data"):
        #     data_param_name = "files"
        #     data  = ...

        # Fallback to use raw text if nothing else worked out
        if data is None:
            data_param_name = "data"
            data = request.body

    return request, data, data_param_name, headers


# tokenize used for automatically fill data param of request
def convert_single_http_requests(
    flow: FlowDetail,
    item_index: int,
    tokenize: bool = True,
    use_requests_session: bool = False,
):
    if not flow.items:
        return "No data"

    request, data, data_param_name, headers = decode_http_request(
        flow.items[item_index].data, tokenize
    )
    if not request.path.startswith("/"):
        raise Exception("request path must start with / to be a valid HTTP request")
    request_path_repr = repr(request.path)
    request_method = validate_request_method(request.command)

    return render(
        HEADER_TEMPLATE,
        use_requests_session=use_requests_session,
        port=flow.port_dst,
    ) + render(
        REQUEST_TEMPLATE,
        headers=repr(headers),
        data=data,
        request_method=request_method,
        request_path_repr=request_path_repr,
        data_param_name=data_param_name,
        use_requests_session=use_requests_session,
        port=flow.port_dst,
        print_info=True,
    )


def convert_flow_to_http_requests(
    flow: FlowDetail, tokenize: bool = True, use_requests_session: bool = True
):
    port = flow.port_dst
    script = render(
        HEADER_TEMPLATE,
        use_requests_session=use_requests_session,
        port=port,
    )

    for item in flow.kind_items():
        if item.direction == "c":
            request, data, data_param_name, headers = decode_http_request(
                item.data, tokenize
            )
            request_method = validate_request_method(request.command)
            if not request.path.startswith("/"):
                raise Exception(
                    "request path must start with / to be a valid HTTP request"
                )
            request_path_repr = repr(request.path)

            script += render(
                REQUEST_TEMPLATE,
                headers=repr(headers),
                data=data,
                request_method=request_method,
                request_path_repr=request_path_repr,
                data_param_name=data_param_name,
                use_requests_session=use_requests_session,
                port=port,
                print_info=True,
            )
    return script


def validate_request_method(request_method: str):
    request_method = request_method.lower()
    if request_method not in [
        "delete",
        "get",
        "head",
        "options",
        "patch",
        "post",
        "put",
    ]:
        # Throw Exception for a bad method to prevent command inject via a nasty request method
        raise Exception(f"Invalid request method: {request_method}")
    return request_method
