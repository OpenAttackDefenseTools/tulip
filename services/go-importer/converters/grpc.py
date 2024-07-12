#!/usr/bin/env python3
from collections import defaultdict
from io import BytesIO
import re
from struct import unpack
from typing import Dict, List
import zlib

from http2 import HTTP2Converter, HeaderTuple
from helpers import Direction, Result, Stream
import hyperframe.frame
from protobuf_inspector.types import StandardParser

# TODO: add license notice from https://github.com/spq/pkappa2
# TODO: Support for gRPC-Web https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md


class GRPCConverter(HTTP2Converter):

    _stream_content_type: Dict[int, Dict[Direction, bool]]
    _stream_responded_grpc_once: bool
    _stream_encoding: Dict[int, Dict[Direction, str]]

    def __init__(self):
        super().__init__()
        self._ansi_escape = re.compile(
            r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')

        self.SETTINGS_NAMES.update({
            65027:
            'GRPC_ALLOW_TRUE_BINARY_METADATA',
            65028:
            'GRPC_PREFERRED_RECEIVE_CRYPTO_FRAME_SIZE'
        })

    def is_valid_encoding(self, encoding: str) -> bool:
        return encoding.lower() in ["identity", "deflate", "gzip"]

    def handle_encoding(self, encoding: str, message_data: bytes) -> bytes:
        encoding = encoding.lower()
        if encoding == "identity":
            return message_data
        elif encoding == "deflate":
            try:
                deflate = zlib.decompressobj(wbits=15)
                return deflate.decompress(message_data)
            except:
                return message_data
        elif encoding == "gzip":
            try:
                deflate = zlib.decompressobj(wbits=15 | 16)
                return deflate.decompress(message_data)
            except:
                return message_data
        else:
            raise ValueError(f"Unknown encoding '{encoding}'")

    def handle_http2_headers(self, direction: Direction,
                             frame: hyperframe.frame.Frame,
                             headers: List[HeaderTuple]) -> None:
        # extract content-type and check if it is grpc
        content_type = next((x[1] for x in headers if x[0] == "content-type"),
                            None)
        if content_type is not None:
            self._stream_content_type[
                frame.stream_id][direction] = content_type.lower() in [
                    "application/grpc", "application/grpc+proto"
                ]
        if self._stream_content_type[frame.stream_id][
                direction] and direction == Direction.SERVERTOCLIENT:
            self._stream_responded_grpc_once = True

        # extract encoding for compression
        # https://github.com/grpc/grpc/blob/master/doc/compression.md
        encoding = next((x[1] for x in headers if x[0] == "grpc-encoding"),
                        None)
        if encoding is not None:
            self._stream_encoding[frame.stream_id][direction] = encoding

    def handle_http2_event(self, direction: Direction,
                           frame: hyperframe.frame.Frame) -> bytes:
        # https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md

        # FIXME: DATA frame boundaries have no relation to Length-Prefixed-Message
        #        boundaries and implementations should make no assumptions about
        #        their alignment.
        #        Do we need to care about this?
        if isinstance(frame, hyperframe.frame.DataFrame):
            # only look at grpc frames
            if frame.stream_id not in self._stream_content_type \
            or not self._stream_content_type[frame.stream_id][direction]:
                # Some servers only send a content-type header in the first
                # response frame in a http2 connection.
                # If we haven't seen a content-type header yet, we assume that
                # the stream is not grpc.
                if direction == Direction.SERVERTOCLIENT and not self._stream_responded_grpc_once:
                    return super().handle_http2_event(direction, frame)

            if len(frame.data) == 0:
                return super().handle_http2_event(direction, frame)

            encoding = "identity"
            if frame.stream_id in self._stream_encoding and direction in self._stream_encoding[
                    frame.stream_id]:
                encoding = self._stream_encoding[frame.stream_id][direction]

            try:
                if len(frame.data) < 5:
                    raise ValueError("Data length is less than 5 bytes")
                output = str(frame).encode()
                compressed_flag = frame.data[0]
                if compressed_flag != 0 and not self.is_valid_encoding(
                        encoding):
                    raise NotImplementedError(
                        "Compressed grpc data is not supported")
                message_length = unpack(">I", frame.data[1:5])[0]
                message_data = frame.data[5:message_length + 5]
                if len(message_data) != message_length:
                    raise ValueError(
                        "Message length does not match the length of the message data"
                    )
                if len(frame.data) > message_length + 5:
                    raise ValueError(
                        "Message data is longer than the message length")
                if compressed_flag != 0:
                    message_data = self.handle_encoding(encoding, message_data)
                output += f"\nGRPC-Compressed: {compressed_flag}\n".encode()
                output += f"GRPC-Message-Length: {message_length}".encode()
                protobuf_message = ""
                if message_data:
                    parser = StandardParser()
                    frame_data = BytesIO(message_data)
                    protobuf_message = parser.parse_message(
                        frame_data, "message")
                output += b"\n" + self._ansi_escape.sub(
                    '', protobuf_message).encode()
                return output + b'\n'
            except Exception as ex:
                return str(ex).encode() + b'\n' + super().handle_http2_event(
                    direction, frame)

        return super().handle_http2_event(direction, frame)

    def handle_stream(self, stream: Stream) -> Result:
        self._stream_content_type = defaultdict(lambda: defaultdict(bool))
        self._stream_responded_grpc_once = False
        self._stream_encoding = defaultdict(lambda: defaultdict(str))
        return super().handle_stream(stream)


if __name__ == "__main__":
    GRPCConverter().run()
