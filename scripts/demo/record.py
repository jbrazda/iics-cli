#!/usr/bin/env python3
"""Record the README demo as an asciinema v2 cast.

usage: record.py [--masks FILE] [--no-mask] [scenes.json] [out.cast]

Each scene types a command at a fake prompt, runs it in a pseudo-terminal,
optionally sends keystrokes (for TUIs) and records the output. Long API
waits are capped so the recording stays short.

Sensitive text (host names, emails, organization names) is masked after
recording with same-length replacements, so tables stay aligned. The
masking rules are read from masks.local.json next to this script, which is
not committed: the rules name the very strings they hide. See
masks.example.json for the format and README.md for the full workflow.
"""
import argparse
import codecs
import fcntl
import json
import os
import pty
import random
import re
import select
import struct
import sys
import termios
import time

HERE = os.path.dirname(os.path.abspath(__file__))
MAX_GAP = 0.6  # cap for idle gaps while a command runs


class Recorder:
    def __init__(self, cols, rows):
        self.cols, self.rows = cols, rows
        self.events = []  # [time, data]
        self.clock = 0.0

    def emit(self, data, dt=0.0):
        self.clock += dt
        self.events.append([round(self.clock, 4), data])

    def prompt(self):
        self.emit("\x1b[1;36m~/demo\x1b[0m \x1b[1;32m❯\x1b[0m ", 0.05)

    def type_text(self, text, lo=0.03, hi=0.08):
        for ch in text:
            self.emit(ch, random.uniform(lo, hi))

    def comment(self, text):
        self.prompt()
        self.emit("\x1b[2m")
        self.type_text("# " + text, 0.02, 0.045)
        self.emit("\x1b[0m\r\n", 0.5)

    def command(self, shown, cmd, keys, env):
        self.prompt()
        self.clock += 0.4
        self.type_text(shown)
        self.emit("\r\n", 0.35)
        self.run(cmd, keys, env)

    def run(self, cmd, keys, env):
        pid, fd = pty.fork()
        if pid == 0:
            os.environ.update(env)
            os.execvp("bash", ["bash", "-c", cmd])
        fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", self.rows, self.cols, 0, 0))
        dec = codecs.getincrementaldecoder("utf-8")("replace")
        last = [time.time()]

        def pump(seconds):
            end = time.time() + seconds
            while time.time() < end:
                ready, _, _ = select.select([fd], [], [], 0.03)
                if not ready:
                    continue
                try:
                    data = os.read(fd, 65536)
                except OSError:
                    return False
                if not data:
                    return False
                now = time.time()
                self.clock += min(now - last[0], MAX_GAP)
                last[0] = now
                # Answer terminal queries as a dark terminal would.
                if b"\x1b]11;?" in data:
                    os.write(fd, b"\x1b]11;rgb:1e1e/1e1e/2e2e\x1b\\")
                if b"\x1b]10;?" in data:
                    os.write(fd, b"\x1b]10;rgb:cdcd/d6d6/f4f4\x1b\\")
                if b"\x1b[6n" in data:
                    os.write(fd, b"\x1b[%d;1R" % self.rows)
                if b"\x1b[c" in data:
                    os.write(fd, b"\x1b[?62;c")
                text = dec.decode(data)
                if text:
                    self.events.append([round(self.clock, 4), text])
            return True

        alive = pump(0.3)
        for k in keys:
            if not alive:
                break
            before = k.get("before", 0.6)
            alive = pump(before)
            last[0] = time.time()
            self.clock += before
            os.write(fd, k["k"].encode())
            alive = pump(k.get("after", 0.4))
        while alive:
            alive = pump(1.0)
        os.waitpid(pid, 0)
        os.close(fd)


def fit(s, n):
    """Pads or cuts s to exactly n characters."""
    return s[:n].ljust(n)


class Masker:
    """Same-length masking from a rules file (see masks.example.json)."""

    def __init__(self, rules):
        em = rules.get("emails", {})
        self.keep = {k.lower() for k in em.get("keep", [])}
        self.names = {k.lower(): v for k, v in em.get("names", {}).items()}
        self.suffixes = em.get("suffixes", [])
        self.other = em.get("other", "user{n}")
        self.domains = [(re.compile(p), r) for p, r in em.get("domains", {}).items()]
        self.rules = [(re.compile(r["pattern"]), r["replace"]) for r in rules.get("rules", [])]
        self.forbid = [f.lower() for f in rules.get("forbid", [])]
        self.seen = {}

    def email(self, m):
        local, domain = m.group(1), m.group(2)
        if local.lower() not in self.keep:
            suffix = next((s for s in self.suffixes if local.lower().endswith(s)), "")
            base = local[: len(local) - len(suffix)]
            if base.lower() not in self.seen:
                fake = self.names.get(base.lower()) or self.other.format(n=len(self.seen) + 1)
                self.seen[base.lower()] = fake
            local = self.seen[base.lower()] + suffix
        for pat, rep in self.domains:
            domain = pat.sub(rep, domain)
        out = local + "@" + domain
        return fit(out, len(m.group(0))) if len(out) <= len(m.group(0)) else "x" * len(m.group(0))

    def __call__(self, text):
        text = re.sub(r"([A-Za-z0-9._%+-]+)@([A-Za-z0-9.-]+\.[A-Za-z]{2,})", self.email, text)
        for pat, rep in self.rules:
            text = pat.sub(lambda m, rep=rep: fit(m.expand(rep), len(m.group(0))), text)
        return text

    def check(self, text):
        low = text.lower()
        return [f for f in self.forbid if f in low]


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("scenes", nargs="?", default=os.path.join(HERE, "scenes.json"))
    ap.add_argument("out", nargs="?", default=os.path.join(HERE, "..", "..", "docs", "demo", "iics-demo.cast"))
    ap.add_argument("--masks", default=os.path.join(HERE, "masks.local.json"))
    ap.add_argument("--no-mask", action="store_true", help="record without masking (for local use only)")
    args = ap.parse_args()

    masker = None
    if not args.no_mask:
        if not os.path.exists(args.masks):
            sys.exit(f"masking rules not found: {args.masks}\n"
                     "copy masks.example.json to masks.local.json and fill in your values, or pass --no-mask")
        with open(args.masks) as f:
            masker = Masker(json.load(f))

    with open(args.scenes) as f:
        spec = json.load(f)
    cols, rows = spec.get("width", 140), spec.get("height", 34)
    env = {"TERM": "xterm-256color", "COLORTERM": "truecolor", "COLUMNS": str(cols), "LINES": str(rows)}
    env.update(spec.get("env", {}))

    rec = Recorder(cols, rows)
    for sc in spec["scenes"]:
        if "comment" in sc:
            rec.comment(sc["comment"])
            continue
        rec.command(sc.get("show", sc["cmd"]), sc["cmd"], sc.get("keys", []), env)
        rec.clock += sc.get("pause", 2.5)
    rec.prompt()
    rec.emit("", 2.0)

    if masker:
        # Mask the joined stream, so text split across output chunks is
        # caught, then cut it back at the original chunk boundaries.
        joined = "".join(e[1] for e in rec.events)
        masked = masker(joined)
        assert len(masked) == len(joined), "masking must keep the length"
        left = masker.check(masked)
        if left:
            sys.exit(f"not written: unmasked text remains: {', '.join(left)}")
        pos = 0
        for e in rec.events:
            n = len(e[1])
            e[1] = masked[pos : pos + n]
            pos += n

    os.makedirs(os.path.dirname(os.path.abspath(args.out)), exist_ok=True)
    with open(args.out, "w") as f:
        f.write(json.dumps({"version": 2, "width": cols, "height": rows,
                            "timestamp": int(time.time()), "idle_time_limit": 3,
                            "title": spec.get("title", "iics demo"),
                            "env": {"TERM": "xterm-256color", "SHELL": "/bin/bash"}}) + "\n")
        for t, d in rec.events:
            if d:
                f.write(json.dumps([t, "o", d]) + "\n")
    print(f"wrote {args.out}: {len(rec.events)} events, {rec.clock:.1f}s")


if __name__ == "__main__":
    main()
