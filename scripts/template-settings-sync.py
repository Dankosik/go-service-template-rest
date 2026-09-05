#!/usr/bin/env python3
"""Sync only the two portable harness depth leaves; retain all other JSON bytes."""

import json
from pathlib import Path
import sys


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate JSON key")
        result[key] = value
    return result


def reject_constant(value):
    raise ValueError("non-JSON numeric constant")


# Numeric consumer values are never converted or serialized, avoiding precision loss.
decoder = json.JSONDecoder(object_pairs_hook=unique_object, parse_float=str,
                           parse_constant=reject_constant)
leaves = {
    ".claude/settings.json": ("env", "CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH"),
    ".qwen/settings.json": ("model", "maxSubagentDepth"),
}


def read_settings(path, group, required):
    if path.exists() and not path.is_file():
        raise ValueError("settings must be a regular file")
    text = path.read_bytes().decode("utf-8") if path.exists() else "{}\n"
    data = decoder.decode(text)
    if not isinstance(data, dict) or (group in data and not isinstance(data[group], dict)):
        raise ValueError("settings and managed parent must be objects")
    if required and group not in data:
        raise ValueError("missing source depth")
    return text, data


def member_span(text, start, key):
    """Find a direct object's member using the stdlib decoder's token boundaries."""
    pos = start + 1
    while True:
        pos += len(text[pos:]) - len(text[pos:].lstrip())
        if text[pos] == "}":
            return None
        name, pos = decoder.raw_decode(text, pos)
        pos += len(text[pos:]) - len(text[pos:].lstrip()) + 1  # colon
        pos += len(text[pos:]) - len(text[pos:].lstrip())
        value_start = pos
        _, pos = decoder.raw_decode(text, pos)
        if name == key:
            return value_start, pos
        pos += len(text[pos:]) - len(text[pos:].lstrip())
        if text[pos] == ",":
            pos += 1


def set_member(text, start, key, value):
    span = member_span(text, start, key)
    if span is not None:
        return text[:span[0]] + json.dumps(value) + text[span[1]:]
    comma = "" if text[start + 1:].lstrip().startswith("}") else ","
    return text[:start + 1] + json.dumps(key) + ": " + json.dumps(value) + comma + text[start + 1:]


def main():
    mode, source_root, target_root, entry = sys.argv[1:]
    group, key = leaves[entry]
    _, source = read_settings(Path(source_root) / entry, group, required=True)
    value = source[group].get(key)
    if group == "env":
        valid = (isinstance(value, str) and value.isascii() and value.isdecimal()
                 and bool(value.lstrip("0")))
    else:
        valid = type(value) is int and 1 <= value <= 100
    if not valid:
        raise ValueError("invalid source depth")
    if mode == "source":
        return 0
    path = Path(target_root) / entry
    text, target = read_settings(path, group, required=False)
    current = target.get(group, {}).get(key)
    if type(current) is type(value) and current == value:
        return 0
    if mode == "check":
        print("  M " + entry + " (managed depth)")
        return 1
    if mode == "preflight":
        return 0
    if mode != "apply":
        raise ValueError("unknown mode")
    start = len(text) - len(text.lstrip())
    span = member_span(text, start, group)
    updated = (set_member(text, span[0], key, value) if span is not None
               else set_member(text, start, group, {key: value}))
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(updated.encode("utf-8"))
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, RecursionError):
        # Never include decoder exceptions or settings content: they may hold credentials.
        print("template-settings-sync: invalid or unreadable settings JSON", file=sys.stderr)
        sys.exit(2)
