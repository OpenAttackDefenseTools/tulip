#!/usr/bin/env python3
import re
from io import BytesIO
from helpers import Converter, Result, Stream, StreamChunk
from protobuf_inspector.types import StandardParser

# TODO: add license notice from https://github.com/spq/pkappa2

class ProtobufConverter(Converter):

    def __init__(self):
        super().__init__()
        self._ansi_escape = re.compile(
            r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')

    def handle_stream(self, stream: Stream) -> Result:
        result_data = []
        for chunk in stream.Chunks:
            try:
                parser = StandardParser()
                frame_data = BytesIO(chunk.Content)
                protobuf_message = parser.parse_message(frame_data, "message")
                result_data.append(
                    StreamChunk(
                        chunk.Direction,
                        self._ansi_escape.sub('', protobuf_message).encode()))
            except Exception as ex:
                result_data.append(
                    StreamChunk(
                        chunk.Direction, b'Protobuf ERROR: ' +
                        str(ex).encode() + b'\n' + chunk.Content))
        return Result(result_data)


if __name__ == "__main__":
    ProtobufConverter().run()
