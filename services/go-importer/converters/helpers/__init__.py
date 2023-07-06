# TODO: add license notice from https://github.com/spq/pkappa2

from dataclasses import dataclass
from enum import Enum
from typing import List
import base64
import datetime
import json
import sys


class Protocol(Enum):
    TCP = 0
    UDP = 1


@dataclass
class StreamMetadata:
    StreamID: int
    ClientHost: str
    ClientPort: int
    ServerHost: str
    ServerPort: int
    Protocol: Protocol


class Direction(Enum):
    CLIENTTOSERVER = 0
    SERVERTOCLIENT = 1


@dataclass
class StreamChunk:
    Direction: Direction
    Content: bytes


@dataclass
class Stream:
    Metadata: StreamMetadata
    Chunks: List[StreamChunk]


@dataclass
class Result:
    Chunks: List[StreamChunk]


class Converter:
    """
    Base class for pkappa2 converters.

    Converters are expected to be implemented as a class that inherits from this
    class and implements the handle_stream method. The handle_stream method
    is called for each stream that is passed to the converter. The converter is
    expected to return a Result object that contains the data that should be
    displayed in the UI.
    """

    current_stream_id: int

    def log(self, message: str):
        """
        Log a message to stderr.

        This method can be used to log messages to the UI. The message will be
        displayed in the stderr tab of the UI.

        Can be used for debugging.
        """
        now = datetime.datetime.now().strftime("%d.%b %Y %H:%M:%S")
        print(f'{now} (stream: {self.current_stream_id}): {message}',
              flush=True,
              file=sys.stderr)

    def run(self):
        """
        Run the converter.

        This method goes into an endless loop that parses the input from
        pkappa2 and calls the handle_stream method for each stream. The
        result of the handle_stream method is then written to stdout.
        """
        self.current_stream_id = -1
        # TODO: move this back to a while true loop so we can use this on-demand
        try:
            flow_entry = json.loads(sys.stdin.buffer.readline().decode("utf-8"))
            metadata = StreamMetadata(
                StreamID=0,  # TODO: is this needed?
                ClientHost=flow_entry['Src_ip'],
                ClientPort=flow_entry['Src_port'],
                ServerHost=flow_entry['Dst_ip'],
                ServerPort=flow_entry['Dst_port'],
                Protocol=Protocol.TCP,  # TODO: Is this correct assumption?
            )

            self.current_stream_id = metadata.StreamID
            stream_chunks = []
            for chunk in flow_entry['Flow']:
                stream_chunks.append(StreamChunk(
                    Content=chunk['Data'].encode(),  # TODO: switch to b64 decoding
                    Direction=Direction.CLIENTTOSERVER if chunk['From'] == 'c' else Direction.SERVERTOCLIENT,
                ))

            stream = Stream(metadata, stream_chunks)
            result = self.handle_stream(stream)

            formatted_chunks = []
            for chunk in result.Chunks:
                formatted_chunks.append({
                    'from': 'c' if chunk.Direction == Direction.CLIENTTOSERVER else 's',
                    'base64_content': base64.b64encode(chunk.Content).decode(),
                })

            json.dump(formatted_chunks, sys.stdout)
            sys.stdout.flush()
        except KeyboardInterrupt:
            return

    def handle_stream(self, stream: Stream) -> Result:
        """
        Transform the data of a stream and return the changed stream.
        The stream contains metadata of the source and target and a list of
        chunks of data. Each chunk contains the direction of the data and the
        data itself. The data is a byte array.

        This method is called for each stream that is passed to the converter.

        Args:
            stream: The stream to transform.

        Returns:
            A Result object that contains the data that should be displayed
            in the UI.
        """
        raise NotImplementedError
