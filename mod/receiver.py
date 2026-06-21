# Receiver de teste do M1 — roda no PC do jogo: python receiver.py
# Le o arquivo que o mod escreve e mostra x,y,z,yaw em tempo real.
import os, time

f = os.path.join(os.environ.get("PUBLIC", r"C:\Users\Public"), "palproxvoice_pos.txt")
print("lendo", f, "- entra no jogo e anda")
while True:
    try:
        with open(f) as fh:
            line = fh.read().strip()
        if line:
            print("\r" + line.ljust(40), end="", flush=True)
    except FileNotFoundError:
        pass
    time.sleep(0.1)
