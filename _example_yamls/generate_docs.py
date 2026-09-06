#!/usr/bin/env python3
"""Convert OCaml .ml files to Markdown.

Doc comments (** ... *) become prose.
Everything else becomes fenced OCaml code blocks.

Usage:
    python ml_to_md.py file.ml              # writes file.md
    python ml_to_md.py file.ml -o out.md    # writes to specified path
    python ml_to_md.py *.ml                 # converts multiple files
"""

import sys
import argparse
from pathlib import Path


def split_segments(source):
    segments = []
    i = 0
    n = len(source)

    while i < n:
        if source[i:i+3] == '(**':
            # Doc comment
            depth = 1
            j = i + 3
            while j < n and depth > 0:
                if source[j:j+2] == '(*':
                    depth += 1
                    j += 2
                elif source[j:j+2] == '*)':
                    depth -= 1
                    j += 2
                else:
                    j += 1
            segments.append(('doc', source[i+3:j-2]))
            i = j

        elif source[i:i+2] == '(*':
            # Regular comment — keep as code
            depth = 1
            j = i + 2
            while j < n and depth > 0:
                if source[j:j+2] == '(*':
                    depth += 1
                    j += 2
                elif source[j:j+2] == '*)':
                    depth -= 1
                    j += 2
                else:
                    j += 1
            segments.append(('code', source[i:j]))
            i = j

        else:
            start = i
            while i < n and not (source[i:i+3] == '(**' or source[i:i+2] == '(*'):
                i += 1
            if source[start:i]:
                segments.append(('code', source[start:i]))

    return segments


# def clean_doc(content):
#     lines = content.split('\n')
#     cleaned = []
#     for line in lines:
#         stripped = line.strip()
#         if stripped.startswith('*'):
#             stripped = stripped[1:].lstrip()
#         cleaned.append(stripped)
#     while cleaned and not cleaned[0]:
#         cleaned.pop(0)
#     while cleaned and not cleaned[-1]:
#         cleaned.pop()
#     return '\n'.join(cleaned)

import re

# def clean_doc(content):
#     lines = content.split('\n')
#     cleaned = []
#     for line in lines:
#         stripped = line.strip()
#         if stripped.startswith('*'):
#             stripped = stripped[1:].lstrip()
#         cleaned.append(stripped)
#     while cleaned and not cleaned[0]:
#         cleaned.pop(0)
#     while cleaned and not cleaned[-1]:
#         cleaned.pop()
#     prose = '\n'.join(cleaned)
#     prose = re.sub(r'\{(\d+) ([^}]+)\}', lambda m: '#' * int(m.group(1)) + ' ' + m.group(2), prose)
#     return prose

def to_anchor(text):
    return re.sub(r'[^\w\s-]', '', text).strip().lower().replace(' ', '-')

def clean_doc(content):
    lines = content.split('\n')
    cleaned = []
    for line in lines:
        stripped = line.strip()
        if stripped.startswith('*'):
            stripped = stripped[1:].lstrip()
        cleaned.append(stripped)
    while cleaned and not cleaned[0]:
        cleaned.pop(0)
    while cleaned and not cleaned[-1]:
        cleaned.pop()
    prose = '\n'.join(cleaned)

    # First pass: collect label -> anchor mappings
    labels = {}
    for m in re.finditer(r'\{(\d+):(\w+) ([^}]+)\}', prose):
        labels[m.group(2)] = to_anchor(m.group(3))

    # Headings with labels
    prose = re.sub(r'\{(\d+):(\w+) ([^}]+)\}',
                   lambda m: '#' * int(m.group(1)) + ' ' + m.group(3), prose)
    # Headings without labels
    prose = re.sub(r'\{(\d+) ([^}]+)\}',
                   lambda m: '#' * int(m.group(1)) + ' ' + m.group(2), prose)
    # Label references
    prose = re.sub(r'\{!(\w+)\}',
                   lambda m: '[{}](#{})'.format(m.group(1), labels.get(m.group(1), m.group(1))), prose)
    # Bold before italics to avoid conflict
    prose = re.sub(r'\{b ([^}]+)\}', r'**\1**', prose)
    prose = re.sub(r'\{i ([^}]+)\}', r'*\1*', prose)
    # Inline code
    prose = re.sub(r'\[([^\]]+)\]', r'`\1`', prose)
    return prose


def clean_code(content):
    lines = content.split('\n')
    while lines and not lines[0].strip():
        lines.pop(0)
    while lines and not lines[-1].strip():
        lines.pop()
    return '\n'.join(lines)


def convert(source):
    parts = []
    for kind, content in split_segments(source):
        if kind == 'doc':
            prose = clean_doc(content)
            if prose:
                parts.append(prose)
        else:
            code = clean_code(content)
            if code:
                parts.append('```ocaml\n' + code + '\n```')
    return '\n\n'.join(parts) + '\n'


def main():
    parser = argparse.ArgumentParser(description='Convert .ml files to .md')
    parser.add_argument('files', nargs='+', help='.ml files to convert')
    parser.add_argument('-o', '--output', help='Output file or directory')
    args = parser.parse_args()

    out = Path(args.output) if args.output else None

    if out and len(args.files) > 1 and out.exists() and not out.is_dir():
        print('Error: -o must be a directory when converting multiple files', file=sys.stderr)
        sys.exit(1)

    for path_str in args.files:
        path = Path(path_str)
        if not path.exists():
            print(f'Error: {path} not found', file=sys.stderr)
            continue
        if path.suffix != '.ml':
            print(f'Warning: skipping {path} (not .ml)', file=sys.stderr)
            continue

        if out is None:
            out_path = path.with_suffix('.md')
        elif out.is_dir():
            out_path = out / path.with_suffix('.md').name
        else:
            out_path = out  # single file, -o is the exact path

        out_path.write_text(convert(path.read_text(encoding='utf-8')), encoding='utf-8')
        print(f'{path} -> {out_path}')


if __name__ == '__main__':
    main()