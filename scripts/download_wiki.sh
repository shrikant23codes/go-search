#!/usr/bin/env bash
set -euo pipefail
mkdir -p data
BASE="https://dumps.wikimedia.org/other/cirrus_search_index/20260510/index_name=enwiki_content"
FILE="enwiki_content-20260510-00010.json.bz2"
echo "Downloading $FILE..."
curl -L "$BASE/$FILE" -o "data/$FILE"
echo "done: $(du -sh data/$FILE)"