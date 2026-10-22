#!/usr/bin/env python3
"""Generate golden test vectors from the Python fairplay-decrypt core.

Usage (from the fairplay-decrypt checkout that provides core.py and the
built transform helper):

    FAIRPLAY_ROOT=/path/to/fairplay-decrypt \
        python scripts/fairplay_vectors.py > third-party/fairplay/vectors_test.go

Requires the fairplay-decrypt virtualenv (cryptography installed).
"""

import hashlib
import os
import random
import sys
from pathlib import Path

ROOT = Path(os.environ.get("FAIRPLAY_ROOT", "../fairplay-decrypt"))
sys.path.insert(0, str(ROOT))

from core import (  # noqa: E402
    BlockTransform,
    derive_page_key,
    derive_work16,
    finish,
    source_blocks,
)
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes  # noqa: E402

random.seed(20261022)


def hexs(b: bytes) -> str:
    return f'"{b.hex()}"'


lines = []
lines.append("package fairplay")
lines.append("")
lines.append("// Code generated from the Python implementation; do not edit by hand.")
lines.append("")
lines.append("var finishVectors = []struct{ state, want string }{")
for _ in range(32):
    state = bytes(random.randrange(256) for _ in range(16))
    lines.append(f"\t{{{hexs(state)}, {hexs(finish(state))}}},")
lines.append("}")

lines.append("")
lines.append("var sourceBlockVectors = []struct{ wrapped string; index uint32; blocks [3]string }{")
for _ in range(16):
    wrapped = bytes(random.randrange(256) for _ in range(16))
    index = random.randrange(1 << 32)
    blocks = source_blocks(wrapped, index)
    lines.append(f"\t{{{hexs(wrapped)}, {index}, [3]string{{{', '.join(hexs(b) for b in blocks)}}}}},")
lines.append("}")

# Synthetic SINF whose priv record decodes with a known credential.
credential = bytes(random.randrange(256) for _ in range(16))
name = b"com.example.app"
iviv = bytes(random.randrange(256) for _ in range(16))
work = bytes(random.randrange(256) for _ in range(16))
iv = hashlib.md5(name + iviv).digest()
clear = b"itun" + b"\x00" * 16 + b"key " + work + b"\x00" * 4 + b"iviv" + b"\x00" * 400
assert len(clear) == 448
key = bytes(x ^ 0x30 for x in credential)
enc = Cipher(algorithms.AES(key), modes.CBC(iv))
priv = enc.encryptor().update(clear) + enc.encryptor().finalize()


def atom(tag: bytes, payload: bytes) -> bytes:
    body = tag + payload
    return (len(body) + 4).to_bytes(4, "big") + body


schi = atom(b"name", name + b"\x00" * 16) + atom(b"iviv", iviv) + atom(b"priv", priv)
sinf = atom(b"sinf", atom(b"schi", schi))

lines.append("")
lines.append("var workVectors = []struct{ credential, sinf, want string }{")
lines.append(f"\t{{{hexs(credential)}, {hexs(sinf)}, {hexs(derive_work16(credential, sinf))}}},")
lines.append("}")

# Page-key vectors via the real C transform helper.
with BlockTransform(ROOT / "transform", ROOT / "transform_const_data.bin") as transform:
    lines.append("")
    lines.append("var pageKeyVectors = []struct{ wrapped string; index uint32; work, want string }{")
    for _ in range(8):
        wrapped = bytes(random.randrange(256) for _ in range(16))
        index = random.randrange(1 << 32)
        w = bytes(random.randrange(256) for _ in range(16))
        lines.append(
            f"\t{{{hexs(wrapped)}, {index}, {hexs(w)}, {hexs(derive_page_key(wrapped, index, w, transform))}}},"
        )
    lines.append("}")

print("\n".join(lines))
