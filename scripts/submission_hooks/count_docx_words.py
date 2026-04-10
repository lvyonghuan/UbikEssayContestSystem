#!/usr/bin/env python3
import json
import re
import sys
import zipfile


def read_docx_text(path):
    with zipfile.ZipFile(path, "r") as docx:
        xml = docx.read("word/document.xml").decode("utf-8", errors="ignore")
    pieces = re.findall(r"<w:t[^>]*>(.*?)</w:t>", xml, flags=re.DOTALL)
    merged = " ".join(pieces)
    merged = re.sub(r"\s+", " ", merged).strip()
    return merged


def count_tokens(text):
    if not text:
        return 0
    # Count contiguous latin/numeric words and each CJK character as one token.
    tokens = re.findall(r"[A-Za-z0-9_]+|[\u4e00-\u9fff]", text)
    if tokens:
        return len(tokens)
    return 1 if text.strip() else 0


def main():
    raw = sys.stdin.read().strip()
    if not raw:
        print(json.dumps({"allow": False, "message": "empty script input"}, ensure_ascii=False))
        return

    data = json.loads(raw)
    payload = data.get("payload") or {}

    saved_path = payload.get("savedPath") or payload.get("saved_path")
    if not saved_path:
        print(json.dumps({"allow": False, "message": "savedPath is required"}, ensure_ascii=False))
        return

    try:
        text = read_docx_text(saved_path)
        word_count = count_tokens(text)
    except Exception as exc:
        print(json.dumps({"allow": False, "message": str(exc)}, ensure_ascii=False))
        return

    print(
        json.dumps(
            {
                "allow": True,
                "patch": {
                    "word_count": word_count,
                },
            },
            ensure_ascii=False,
        )
    )


if __name__ == "__main__":
    main()
