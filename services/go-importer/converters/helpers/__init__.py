# TODO: add license notice from https://github.com/spq/pkappa2

from dataclasses import dataclass
from enum import Enum
from typing import List
import traceback
import datetime
import os
import sys
import msgpack


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
        stdin = os.fdopen(sys.stdin.fileno(), 'rb', buffering=0)
        unpacker = msgpack.Unpacker(stdin, raw=True)

        for data in unpacker:
            try:
                metadata = StreamMetadata(
                    StreamID=0,  # TODO: is this needed?
                    ClientHost=data[b'Src_ip'],
                    ClientPort=data[b'Src_port'],
                    ServerHost=data[b'Dst_ip'],
                    ServerPort=data[b'Dst_port'],
                    Protocol=Protocol.TCP,
                )

                self.current_stream_id = metadata.StreamID
                stream_chunks = []
                for chunk in data[b'Flow']:
                    stream_chunks.append(StreamChunk(
                        Content=chunk[b'Data'],
                        Direction=Direction.CLIENTTOSERVER if chunk[b'From'] == b'c' else Direction.SERVERTOCLIENT,
                    ))

                stream = Stream(metadata, stream_chunks)
                result = self.handle_stream(stream)

                formatted_chunks = []
                for chunk in result.Chunks:
                    formatted_chunks.append({
                        'From': 'c' if chunk.Direction == Direction.CLIENTTOSERVER else 's',
                        'Data': chunk.Content,
                    })

                # Naive implementation of checking if it looks like the output data changed at all, if it seems
                # like it didn't, avoid showing it
                # TODO: is just comparing full payload a good idea? should it be implemented on converter-level instead?
                changed = False
                if len(stream_chunks) != len(formatted_chunks):
                    changed = True
                else:
                    for stream_chunk, formatted_chunk in zip(stream_chunks, formatted_chunks):
                        if len(stream_chunk.Content) != len(formatted_chunk['Data']):
                            changed = True
                            break

                sys.stdout.buffer.write(
                    msgpack.packb(formatted_chunks if changed else [], use_bin_type=True)
                )
                sys.stdout.buffer.flush()
            except KeyboardInterrupt:
                return
            except Exception as e:
                print(f'Ran into an exception: {e}\n{traceback.format_exc()}', file=sys.stderr, flush=True)
                sys.stdout.buffer.write(
                    msgpack.packb([], use_bin_type=True)
                )
                sys.stdout.buffer.flush()

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
