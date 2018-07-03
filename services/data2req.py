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

import pprint

try:
    from BaseHTTPServer import BaseHTTPRequestHandler
    from StringIO import StringIO
except ImportError:
    # python3
    from http.server import BaseHTTPRequestHandler
    from io import StringIO

#class to parse request informations
class HTTPRequest(BaseHTTPRequestHandler):
    def __init__(self, request_text):
        self.rfile = StringIO(request_text)
        self.raw_requestline = self.rfile.readline()
        self.error_code = self.error_message = None
        self.parse_request()

    def send_error(self, code, message):
        self.error_code = code
        self.error_message = message

# tokenize used for automatically fill data param of request
def convert_http_requests(data, tokenize=True):
    request = HTTPRequest(data)
    body = data.split("\n\n", 1)

    tokens = {}
    headers = {}

    if tokenize and len(body) > 1:
        for i in body[1].split("&"):
            d = i.split("=")
            tokens[d[0]] = d[1]

    blocked_headers = ["content-length", "accept-encoding", "connection", "accept"]

    for i in request.headers:
        if not i in blocked_headers:
            headers[i] = request.headers[i]

    return """requests.{}("http://"+sys.argv[1]+"{}",\n\tdata={},\n\theaders={}\n)""".format(
        request.command.lower(),
        request.path,
        tokens,
        str(dict(headers)),
    )




test_data = """GET /messages HTTP/1.1
User-Agent: Mozilla/5.0 (Unknown; Linux i686) AppleWebKit/534.34 (KHTML, like Gecko) PhantomJS/1.9.8 Safari/534.34
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
Cookie: session=.eJwdjsFqwzAQBX-l7DkULEcXQ6EHucGHXZMgW0iX0NhOXMlKQG5QrZB_r9vjg5nhPeB4DsM8QvEd7sMGjl89FA94OUEBJFpXS2ONLCMJx1A0kRhynfCnliXT_jCRpdHIC0PWWrKTJ4lRq32qVclRVAvJMmFq-OrlpJCtbDRCs1o1SdsmQ9l6s9tz3GEi1fpadFta919XW-PJVzmmbtEWOcmPkWSVa-syTHoxtkzkdWaEe4PnBu7zEP7_wzj769KncIm8f8_c-XPqbtfXMJzg-QtOmlBt.DhJ0zQ.EgQaH_4t3viAFoeSsir_tVxdBDo
Connection: Keep-Alive
Accept-Encoding: gzip
Accept-Language: en-US,*
Host: 10.0.1.1:5010"""

if __name__ == "__main__":
    print(convert_http_requests(data, False))
