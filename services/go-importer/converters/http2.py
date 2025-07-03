#!/usr/bin/env python3
from base64 import urlsafe_b64decode
from collections import defaultdict
from typing import Dict, List, Optional
import h2.frame_buffer
from h2.exceptions import H2Error
import hyperframe.frame
from hpack import Decoder, HeaderTuple
from http_gzip import HTTPConverter, HTTPRequest, HTTPResponse
from helpers import StreamChunk, Direction, Result, Stream

# TODO: add license notice from https://github.com/spq/pkappa2
# pip install h2


class HTTP2Converter(HTTPConverter):

    SETTINGS_NAMES = {
        1: "HEADER_TABLE_SIZE",
        2: "ENABLE_PUSH",
        3: "MAX_CONCURRENT_STREAMS",
        4: "INITIAL_WINDOW_SIZE",
        5: "MAX_FRAME_SIZE",
        6: "MAX_HEADER_LIST_SIZE",
    }

    hpack_decoder: Dict[Direction, Decoder]

    def __init__(self):
        super().__init__()
        self.h2_client_buffer = None
        self.h2_server_buffer = None
        self.h2_active = False

    def handle_http2_event(self, direction: Direction,
                           frame: hyperframe.frame.Frame) -> bytes:
        return self.format_http2_frame(direction, frame)

    # Avoid decoding the headers multiple times.
    def handle_http2_headers(self, direction: Direction,
                             frame: hyperframe.frame.Frame,
                             headers: List[HeaderTuple]) -> None:
        pass

    def format_http2_frame(self, direction: Direction,
                           frame: hyperframe.frame.Frame) -> bytes:
        if isinstance(
                frame,
            (hyperframe.frame.HeadersFrame, hyperframe.frame.PushPromiseFrame,
             hyperframe.frame.ContinuationFrame)):
            if 'END_HEADERS' not in frame.flags:
                raise Exception('TODO: Handle fragmented headers')
            headers = self.hpack_decoder[direction].decode(frame.data)
            self.handle_http2_headers(direction, frame, headers)
            output = ''
            for header in headers:
                output += f"{header[0]}: {header[1]}\n"
            return f"{frame}\n{output}".encode()
        elif isinstance(frame, hyperframe.frame.DataFrame):
            return str(frame).encode() + b"\n" + frame.data + b"\n"
        elif isinstance(frame, hyperframe.frame.SettingsFrame):
            settings = {
                self.SETTINGS_NAMES.get(k, k): v
                for k, v in frame.settings.items()
            }
            # TODO: Update hpack decoder MAX_HEADER_LIST_SIZE with setting. The values only take effect after an ACK.
            output = ''
            for k, v in settings.items():
                output += f"{k}: {v}\n"
            return f"{type(frame).__name__}(stream_id={frame.stream_id}, flags={frame.flags!r}):\n{output}".encode(
            )
        return (str(frame) + "\n").encode()

    def setup_http2_buffers(self):
        self.h2_server_buffer = h2.frame_buffer.FrameBuffer(server=True)
        self.h2_server_buffer.max_frame_size = 16384
        self.h2_client_buffer = h2.frame_buffer.FrameBuffer(server=False)
        self.h2_client_buffer.max_frame_size = 16384

    def handle_http2_upgrade(self, request: HTTPRequest) -> List[StreamChunk]:
        self.setup_http2_buffers()
        settings = request.headers.get("HTTP2-Settings")
        if settings:
            f = hyperframe.frame.SettingsFrame(0)
            f.parse_body(urlsafe_b64decode(settings))
            return [
                StreamChunk(
                    Direction.CLIENTTOSERVER,
                    self.handle_http2_event(Direction.CLIENTTOSERVER, f) +
                    b"\n")
            ]
        return []

    def handle_http2_init(self, chunk: bytes) -> List[StreamChunk]:
        self.setup_http2_buffers()
        self.h2_active = True
        return self.handle_http2_request(chunk)

    def handle_http2_request(self, chunk: bytes) -> List[StreamChunk]:
        if not self.h2_server_buffer:
            return [StreamChunk(Direction.CLIENTTOSERVER, chunk)]
        self.h2_server_buffer.add_data(chunk)
        # TODO: Update max_frame_size when observing SETTINGS frames updating it
        events = []
        for event in self.h2_server_buffer:
            events.append(
                StreamChunk(
                    Direction.CLIENTTOSERVER,
                    self.handle_http2_event(Direction.CLIENTTOSERVER, event) +
                    b"\n"))
        return events

    def handle_http2_response(self, chunk: bytes) -> List[StreamChunk]:
        if not self.h2_client_buffer:
            return [StreamChunk(Direction.SERVERTOCLIENT, chunk)]
        self.h2_active = True
        self.h2_client_buffer.add_data(chunk)
        events = []
        for event in self.h2_client_buffer:
            events.append(
                StreamChunk(
                    Direction.SERVERTOCLIENT,
                    self.handle_http2_event(Direction.SERVERTOCLIENT, event) +
                    b"\n"))
        return events

    def handle_raw_client_chunk(
            self, chunk: StreamChunk) -> Optional[List[StreamChunk]]:
        try:
            if self.h2_active:
                return self.handle_http2_request(chunk.Content)

            if chunk.Content.startswith(b"PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"):
                # HTTP/2
                return self.handle_http2_init(chunk.Content)
        except H2Error as ex:
            data = f"Unable to parse HTTP2 init request: {ex}".encode()
            return [StreamChunk(chunk.Direction, data)]
        # continue parsing HTTP/1 request
        return super().handle_raw_client_chunk(chunk)

    def handle_raw_server_chunk(
            self, chunk: StreamChunk) -> Optional[List[StreamChunk]]:
        if self.h2_active:
            # HTTP/2
            try:
                return self.handle_http2_response(chunk.Content)
            except H2Error as ex:
                data = f"Unable to parse HTTP2 response: {ex}".encode()
                return [StreamChunk(chunk.Direction, data)]
        # continue parsing HTTP/1 response
        return super().handle_raw_server_chunk(chunk)

    def handle_http1_request(self, chunk: StreamChunk,
                             request: HTTPRequest) -> List[StreamChunk]:

        # https://httpwg.org/specs/rfc7540.html#discover-http
        connection = request.headers.get("Connection")
        if connection:
            connection_headers = list(
                map(lambda h: h.strip(), connection.split(",")))
            if "Upgrade" in connection_headers and request.headers.get(
                    "Upgrade") == "h2c":
                # HTTP/2
                return [chunk] + self.handle_http2_upgrade(request)

        return super().handle_http1_request(chunk, request)

    def handle_http1_response(self, header: bytes, body: bytes,
                              chunk: StreamChunk,
                              response: HTTPResponse) -> List[StreamChunk]:

        if response.headers.get(
                "Connection") == "Upgrade" and response.headers.get(
                    "Upgrade") == "h2c":
            # HTTP/2
            if self.h2_server_buffer is None:
                raise Exception("HTTP/2 upgrade request not found")

            return [StreamChunk(chunk.Direction, header + b'\r\n\r\n')
                    ] + self.handle_http2_response(response.data)

        return super().handle_http1_response(header, body, chunk, response)

    def handle_stream(self, stream: Stream) -> Result:
        self.h2_active = False
        self.h2_client_buffer = None
        self.hpack_decoder = defaultdict(Decoder)
        self.h2_server_buffer = None
        if len(stream.Chunks) > 1 and stream.Chunks[
                0].Direction == Direction.SERVERTOCLIENT:
            # the server send something before the request arrived?!
            # try to fix it by switching the first and second chunk transparently.
            return super().handle_stream(
                Stream(
                    stream.Metadata, stream.Chunks[1:2] + stream.Chunks[0:1] +
                    stream.Chunks[2:]))
        return super().handle_stream(stream)


if __name__ == "__main__":
    HTTP2Converter().run()
