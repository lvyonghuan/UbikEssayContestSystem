#!/usr/bin/env python3
"""Count words from a saved DOCX file and return patch JSON for script flow."""

import json
import os
import re
import sys
import zipfile
from xml.etree import ElementTree

CJK = re.compile(
    r"[\u1100-\u11ff\u2e80-\u9fff\ua000-\ua4ff\uac00-\ud7af"
    r"\uf900-\ufaff\ufe30-\ufe4f"
    r"\U00020000-\U0002a6df]"
)

WORD_NS = "{http://schemas.openxmlformats.org/wordprocessingml/2006/main}"
WORD_DOCUMENT_XML = "word/document.xml"


def emit(payload: dict) -> None:
    sys.stdout.write(json.dumps(payload))


def block(message: str) -> None:
    emit({"allow": False, "message": message})


def parse_execute_input(raw: str) -> dict:
    data = json.loads(raw)
    if not isinstance(data, dict):
        raise ValueError("stdin input must be a JSON object")
    return data


def resolve_saved_docx_path(data: dict) -> str:
    payload = data.get("payload")
    if not isinstance(payload, dict):
        raise ValueError("payload is required")

    saved_path = payload.get("savedPath")
    if not isinstance(saved_path, str) or not saved_path.strip():
        raise ValueError("payload.savedPath is required")

    normalized = os.path.normpath(saved_path.strip())
    if not normalized.lower().endswith(".docx"):
        raise ValueError("payload.savedPath must point to a .docx file")

    if not os.path.isabs(normalized):
        normalized = os.path.normpath(os.path.join(os.getcwd(), normalized))

    if not os.path.isfile(normalized):
        raise FileNotFoundError("docx file not found: " + normalized)

    return normalized


def extract_docx_text(docx_path: str) -> str:
    try:
        with zipfile.ZipFile(docx_path, "r") as zf:
            xml_bytes = zf.read(WORD_DOCUMENT_XML)
    except KeyError as exc:
        raise ValueError("invalid docx: missing word/document.xml") from exc

    root = ElementTree.fromstring(xml_bytes)
    chunks = []

    for node in root.iter():
        if node.tag == WORD_NS + "t" and node.text:
            chunks.append(node.text)
        elif node.tag in (WORD_NS + "tab", WORD_NS + "br", WORD_NS + "cr", WORD_NS + "p"):
            chunks.append(" ")

    return "".join(chunks)


def count_words_word_style(text: str) -> int:
    try:
        import icu  # type: ignore
    except Exception as exc:
        raise RuntimeError("PyICU is required for count_docx_words.py") from exc

    count = 0
    parts = CJK.split(text)
    cjk_chars = CJK.findall(text)
    count += len(cjk_chars)

    bi = icu.BreakIterator.createWordInstance(icu.Locale("en_US"))
    for part in parts:
        if not part.strip():
            continue

        bi.setText(part)
        prev = bi.first()
        pos = bi.nextBoundary()

        while pos != icu.BreakIterator.DONE:
            token = part[prev:pos]
            if bi.getRuleStatus() != 0 and token.strip():
                count += 1
            prev = pos
            pos = bi.nextBoundary()

    return count


def main() -> int:
    raw = sys.stdin.read()
    if not raw.strip():
        block("empty stdin")
        return 0

    try:
        data = parse_execute_input(raw)
        docx_path = resolve_saved_docx_path(data)
        text = extract_docx_text(docx_path)
        word_count = count_words_word_style(text)
    except Exception as exc:
        block(str(exc))
        return 0

    emit({"allow": True, "patch": {"word_count": int(word_count)}})
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
