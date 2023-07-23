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

#class to parse request informations
class HTTPRequest(BaseHTTPRequestHandler):
    def __init__(self, raw_http_request):
        self.rfile = BytesIO(raw_http_request)
        self.raw_requestline = self.rfile.readline()
        self.error_code = self.error_message = None
        self.parse_request()

        try:
            self.headers = dict(self.headers)
        except AttributeError:
            self.headers = {}

        # Data
        try:
            self.body = raw_http_request.split(b"\r\n\r\n", 1)[1].rstrip()
        except IndexError:
            self.body = None

    def send_error(self, code, message):
        self.error_code = code
        self.error_message = message

def decode_http_request(raw_request, tokenize):
    request = HTTPRequest(raw_request)

    data = {}
    headers = {}

    blocked_headers = ["content-length", "accept-encoding", "connection", "accept", "host"]
    content_type = ""
    data_param_name = "data"
    body = request.body

    for i in request.headers:
        normalized_header = i.lower()

        if normalized_header == "content-type":
            content_type = request.headers[i]
        if not normalized_header in blocked_headers:
            headers[i] = request.headers[i]

    # if tokenization is enabled and body is not empty, try to decode form body or JSON body
    if tokenize and body:
        # try to deserialize form data
        if content_type.startswith("application/x-www-form-urlencoded"):
            data_param_name = "data"
            body_dict = parse_qs(body.decode())
            for key, value in body_dict.items():
                if len(value) == 1:
                    data[key] = value[0]
                else:
                    data[key] = value

        # try to deserialize json
        if content_type.startswith("application/json"):
            data_param_name = "json"
            data = json.loads(body)

        # try to use raw text
        if content_type.startswith("text/plain"):
            data_param_name = "data"
            data = body

        # try to extract files
        if content_type.startswith("multipart/form-data"):
            data_param_name = "files"
            return "Forms with files are not yet implemented", None, None, None
    return request, data, data_param_name, headers

# tokenize used for automatically fill data param of request
def convert_single_http_requests(raw_request, flow, tokenize=True, use_requests_session=False):

    request, data, data_param_name, headers = decode_http_request(raw_request, tokenize)
    if not request.path.startswith('/'):
        raise Exception('request path must start with / to be a valid HTTP request')
    request_path_repr = repr(request.path)
    request_method = validate_request_method(request.command)

    rtemplate = Environment(loader=BaseLoader()).from_string("""import os
import requests
import sys

host = sys.argv[1]
{% if use_requests_session %}
s = requests.Session()

s.headers = {{headers}}
{% else %}
headers = {{headers}}
{% endif %}
data = {{data}}

{% if use_requests_session %}s{% else %}requests{% endif %}.{{request_method}}(f"http://{{ '{' }}host{{ '}' }}:{{port}}" + {{request_path_repr}}, {{data_param_name}}=data{% if not use_requests_session %}, headers=headers{% endif %})""")

    return rtemplate.render(
            headers=str(dict(headers)),
            data=data,
            request_method=request_method,
            request_path_repr=request_path_repr,
            data_param_name=data_param_name,
            use_requests_session=use_requests_session,
            port=flow["dst_port"]
        )


def render(template, **kwargs):
    return Environment(loader=BaseLoader()).from_string(template).render(kwargs)

def convert_flow_to_http_requests(flow, tokenize=True, use_requests_session=True):
    port = flow["dst_port"]
    script = render("""import os
import requests
import sys

host = sys.argv[1]
{% if use_requests_session %}
s = requests.Session()
{% endif %}""",use_requests_session=use_requests_session,
            port=port)
    for message in flow['flow']:
        if message['from'] == 'c':
            request, data, data_param_name, headers = decode_http_request(message['data'].encode(), tokenize)
            request_method = validate_request_method(request.command)
            if not request.path.startswith('/'):
                raise Exception('request path must start with / to be a valid HTTP request')
            request_path_repr = repr(request.path)

            script += render("""
{% if use_requests_session %}
s.headers = {{headers}}
{% else %}
headers = {{headers}}
{% endif %}
data = {{data}}
{% if use_requests_session %}s{% else %}requests{% endif %}.{{request_method}}(f"http://{{ '{' }}host{{ '}' }}:{{port}}" + {{request_path_repr}}, {{data_param_name}}=data{% if not use_requests_session %}, headers=headers{% endif %})""",
            headers=str(dict(headers)),
            data=data,
            request_method=request_method,
            request_path_repr=request_path_repr,
            data_param_name=data_param_name,
            use_requests_session=use_requests_session,
            port=port)
    return script

def validate_request_method(request_method: str):
    request_method = request_method.lower()
    if request_method not in ['delete', 'get', 'head', 'options', 'patch', 'post', 'put']:
        # Throw Exception for a bad method to prevent command inject via a nasty request method
        raise Exception(f'Invalid request method: {request_method}')
    return request_method
