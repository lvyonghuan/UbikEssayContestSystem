#!/usr/bin/env python3
import json
import sys


def to_int(value, default):
    try:
        if value is None:
            return default
        return int(value)
    except (TypeError, ValueError):
        return default


def main():
    raw = sys.stdin.read().strip()
    if not raw:
        print(json.dumps({"allow": True}, ensure_ascii=False))
        return

    data = json.loads(raw)
    payload = data.get("payload") or {}
    context = data.get("context") or {}
    step_input = context.get("stepInput") or {}

    max_count = to_int(step_input.get("maxCount"), 3)
    if max_count <= 0:
        max_count = 3

    existing_count = to_int(payload.get("existingCount"), 0)

    if existing_count >= max_count:
        print(
            json.dumps(
                {
                    "allow": False,
                    "message": f"at most {max_count} submissions are allowed in this contest",
                },
                ensure_ascii=False,
            )
        )
        return

    print(json.dumps({"allow": True}, ensure_ascii=False))


if __name__ == "__main__":
    main()
