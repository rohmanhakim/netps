import socket
import select
import sys
import os

# Define ports to listen on
PORTS = [8080, 8081, 8082]
LISTENING_SOCKETS = []

for port in PORTS:
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind(('', port))
        s.listen(5)
        LISTENING_SOCKETS.append(s)
        print(f"Process {s.getsockname()[0]}:{port} (PID {os.getpid()}) is listening")
    except socket.error as e:
        print(f"Could not start listening on port {port}: {e}")
        sys.exit(1)

# Use select to manage multiple sockets
while True:
    read_sockets, _, _ = select.select(LISTENING_SOCKETS, [], [])
    for sock in read_sockets:
        conn, addr = sock.accept()
        print(f"Connection from {addr} on port {sock.getsockname()[1]}")
        # Handle the connection (e.g., in a new thread or process in a real app,
        # but here we just simulate for demonstration)
        conn.sendall(b"Hello from single PID server!")
        conn.close()

