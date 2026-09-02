#!/bin/bash
find . -maxdepth 1 -name "*.md" | while read -r f; do C="$(basename "${f}" .md|tr "[:upper:]" "[:lower:]")"; echo "- [$C](./${f})"; done >> INDEX.md
