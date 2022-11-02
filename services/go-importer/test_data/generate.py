from pwn import *
import threading

l = listen(1337)
r = remote("localhost", 1337)

r.sendline(b"This should be searchable")
l.sendlineafter(b"This should be searchable", b"And so should this line")

def listen(p):
    p.recvall()
def send(p):
    g = cyclic_gen(n=8)
    payload = g.get(6*1024*1024)
    p.sendlineafter(b"line", payload)

sender   = threading.Thread(target=send, args=(r,))
listener = threading.Thread(target=listen, args=(l,))
sender.start()
listener.start()


print("sent")


sender.join()
listener.join()

