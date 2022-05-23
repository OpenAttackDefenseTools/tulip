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
from urllib.parse import urlparse, parse_qs
from io import BytesIO

#class to parse request informations
class HTTPRequest(BaseHTTPRequestHandler):
    def __init__(self, raw_http_request):
        self.rfile = BytesIO(raw_http_request)
        self.raw_requestline = self.rfile.readline()
        self.error_code = self.error_message = None
        self.parse_request()

        self.headers = dict(self.headers)
        # Data
        try:
            self.data = raw_http_request[raw_http_request.index(
                b'\r\n\r\n')+2:].rstrip()
        except ValueError:
            self.data = None

    def send_error(self, code, message):
        self.error_code = code
        self.error_message = message

# tokenize used for automatically fill data param of request
def convert_http_requests(data, tokenize=True):
    request = HTTPRequest(data)

    params = {}
    headers = {}

    if tokenize:
        query_dict = parse_qs(urlparse(request.path).query)
        for key, value in query_dict.items():
            params[key] = value[0]

    blocked_headers = ["content-length", "accept-encoding", "connection", "accept"]

    for i in request.headers:
        if not i.lower() in blocked_headers:
            headers[i] = request.headers[i]

    # TODO; use proper templates instead of format strings.
    # This is already a little clunky, since we want to support at least POST/GET in the
    # same template
    return """import sys
import requests

host = sys.argv[1]

headers = {}

params = {}

data = {}

requests.{}("http://{{}}{}".format(host), params=params, headers=headers, data=data)""".format(
        str(dict(headers)),
        params,
        request.data,
        request.command.lower(),
        request.path,
    )
