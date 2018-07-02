#convert a flow into pwn script
def flow2pwn(flow):
    ip = flow["dst_ip"]
    port = flow["dst_port"]

    script = """from pwn import *

proc = remote('{}', {})
""".format(ip, port)

    for message in flow['flow']:
        if message['from'] == 's':
            script += """proc.writeline("{}")\n""".format(message['data'][:-1])

        else:
            for m in range(len(message['data'])):
                script += """proc.recvuntil("{}")\n""".format(message['data'][-20:].replace("\n","\\n"))
                break

    return script
