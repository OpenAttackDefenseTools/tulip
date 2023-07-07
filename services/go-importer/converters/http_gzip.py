#!/usr/bin/env python3
from http.server import BaseHTTPRequestHandler
from http.client import parse_headers
from io import BytesIO
from typing import List, Optional
from urllib3 import HTTPResponse
from helpers import Converter, StreamChunk, Direction, Result, Stream
import traceback

# TODO: add license notice from https://github.com/spq/pkappa2
# https://stackoverflow.com/questions/4685217/parse-raw-http-headers
class HTTPRequest(BaseHTTPRequestHandler):

    def __init__(self, request_text: str):
        self.rfile = BytesIO(request_text)
        self.raw_requestline = self.rfile.readline()
        self.error_code = self.error_message = None
        self.parse_request()

    def send_error(self, code, message):
        self.error_code = code
        self.error_message = message


class HTTPConverter(Converter):

    def handle_raw_client_chunk(
            self, chunk: StreamChunk) -> Optional[List[StreamChunk]]:
        """
        Handle raw client chunk. Return None to continue parsing HTTP1 request.

        Args:
            chunk (StreamChunk): Raw client chunk.
        
        Returns:
            Optional[List[StreamChunk]]: List of chunks to be added to the result.
        """
        return None

    def handle_raw_server_chunk(
            self, chunk: StreamChunk) -> Optional[List[StreamChunk]]:
        return None

    def handle_http1_request(self, chunk: StreamChunk,
                             request: HTTPRequest) -> List[StreamChunk]:

        # Just pass HTTP1 requests through untouched
        return [StreamChunk(chunk.Direction, chunk.Content)]

    def handle_http1_response(self, header: bytes, body: bytes,
                              chunk: StreamChunk,
                              response: HTTPResponse) -> List[StreamChunk]:

        data = header + b"\r\n\r\n" + response.data
        return [StreamChunk(chunk.Direction, data)]

    def handle_stream(self, stream: Stream) -> Result:

        result_data = []
        for chunk in stream.Chunks:
            if chunk.Direction == Direction.CLIENTTOSERVER:

                raw_result: Optional[
                    List[StreamChunk]] = self.handle_raw_client_chunk(chunk)
                if raw_result is not None:
                    result_data.extend(raw_result)
                    continue

                try:
                    request = HTTPRequest(chunk.Content)
                    if request.error_code:
                        raise Exception(
                            f"{request.error_code} {request.error_message}".
                            encode())

                    result_data.extend(
                        self.handle_http1_request(chunk, request))
                except Exception as ex:
                    data = f"Unable to parse HTTP request: {ex}".encode()
                    result_data.append(StreamChunk(chunk.Direction, data))
            else:
                try:
                    raw_response: Optional[List[
                        StreamChunk]] = self.handle_raw_server_chunk(chunk)
                    if raw_response is not None:
                        result_data.extend(raw_response)
                        continue

                    # https://stackoverflow.com/a/52418392
                    header, body = chunk.Content.split(b"\r\n\r\n", 1)
                    header_stream = BytesIO(header)
                    requestline = header_stream.readline().split(b' ')
                    status = int(requestline[1])
                    headers = parse_headers(header_stream)

                    # tulip: fix the header format, HTTPResponse expects a dict?
                    headers = headers.items()

                    body_stream = BytesIO(body)
                    response = HTTPResponse(
                        body=body_stream,
                        headers=headers,
                        status=status,
                    )

                    result_data.extend(
                        self.handle_http1_response(header, body, chunk,
                                                   response))
                except Exception as ex:
                    data = f"Unable to parse HTTP response: {ex}\n{traceback.format_exc()}".encode()
                    result_data.append(StreamChunk(chunk.Direction, data))
        return Result(result_data)


if __name__ == "__main__":
    HTTPConverter().run()
