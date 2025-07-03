#!/usr/bin/env python3
from typing import List
from http_gzip import HTTPConverter, HTTPRequest, HTTPResponse
from helpers import Direction, Result, Stream, StreamChunk

# TODO: add license notice from https://github.com/spq/pkappa2

class PythonRequestsConverter(HTTPConverter):

    requests_output: str
    target_host: str

    SHORTCUT_METHODS = ["get", "post", "put", "delete", "head", "patch"]

    def handle_http1_request(self, chunk: StreamChunk,
                             request: HTTPRequest) -> List[StreamChunk]:

        data = request.rfile.read()
        headers = {}
        for k, v in request.headers.items():
            headers[k] = v
        if request.command.lower() in self.SHORTCUT_METHODS:
            self.requests_output += f'r = s.{request.command.lower()}('
        else:
            self.requests_output += f'r = s.request({request.command!r}, '
        self.requests_output += f'f"http://{self.target_host}{request.path}"'
        if len(headers) > 0:
            self.requests_output += f', headers={headers}'
        if len(data) > 0:
            self.requests_output += f', data={data}'
        self.requests_output += ')\n'

        return []

    # ignore responses
    def handle_http1_response(self, header: bytes, body: bytes,
                              chunk: StreamChunk,
                              response: HTTPResponse) -> List[StreamChunk]:
        return []

    def handle_stream(self, stream: Stream) -> Result:
        self.requests_output = f'''#!/usr/bin/env python3
import requests
import sys

IP = '{stream.Metadata.ServerHost}'
# IP = sys.argv[1]

# Generated from stream {stream.Metadata.StreamID}
s = requests.Session()

'''
        port = ''
        if stream.Metadata.ServerPort != 80:
            port = f':{stream.Metadata.ServerPort}'
        self.target_host = f'{{IP}}{port}'
        result = super().handle_stream(stream)

        return Result(result.Chunks + [
            StreamChunk(Direction.CLIENTTOSERVER,
                        self.requests_output.encode())
        ])


if __name__ == "__main__":
    PythonRequestsConverter().run()
