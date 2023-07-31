#!/usr/bin/env python3
from helpers import Converter, StreamChunk, Result, Stream, Direction, Protocol

# TODO: add license notice from https://github.com/spq/pkappa2

class PwntoolsRemoteConverter(Converter):

    def handle_stream(self, stream: Stream) -> Result:
        typ = ''
        if stream.Metadata.Protocol == Protocol.UDP:
            typ = ', typ = "udp"'
        output = f'''#!/usr/bin/env python3
from pwn import *
import sys

# Generated from stream {stream.Metadata.StreamID}
# io = remote(sys.argv[1], {stream.Metadata.ServerPort}{typ})
io = remote({stream.Metadata.ServerHost!r}, {stream.Metadata.ServerPort}{typ})
'''
        for i, chunk in enumerate(stream.Chunks):
            if chunk.Direction == Direction.CLIENTTOSERVER:
                if chunk.Content[-1:] == b'\n':
                    output += f"io.sendline({chunk.Content[:-1]!r})\n"
                else:
                    output += f"io.send({chunk.Content!r})\n"
            else:
                if i == len(stream.Chunks) - 1:
                    output += "io.stream()\n"
                else:
                    output += f"io.recvuntil({chunk.Content[-20:]!r})\n"
        if len(stream.Chunks) > 0 and stream.Chunks[
                -1].Direction == Direction.CLIENTTOSERVER:
            output += "io.interactive()\n"
        return Result([StreamChunk(Direction.CLIENTTOSERVER, output.encode())])


if __name__ == "__main__":
    PwntoolsRemoteConverter().run()
