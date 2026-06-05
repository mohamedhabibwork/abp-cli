#!/usr/bin/env sh
set -eu

version="${1:-}"
log_range="${2:-HEAD}"

if [ -z "$version" ]; then
  echo "Usage: scripts/update-changelog.sh <version> [git-log-range]" >&2
  exit 1
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Changelog generation requires a Git checkout." >&2
  exit 1
fi

changelog_file="${CHANGELOG_FILE:-CHANGELOG.md}"
release_notes_file="${RELEASE_NOTES_FILE:-dist/release-notes.md}"
module_path="${MODULE_PATH:-github.com/mohamedhabibwork/abp-cli}"
release_date="${CHANGELOG_DATE:-$(date -u +%Y-%m-%d)}"

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/abp-cli-changelog.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

commits_file="$tmp_dir/commits"
entry_file="$tmp_dir/entry.md"
notes_file="$tmp_dir/release-notes.md"
without_entry_file="$tmp_dir/changelog-without-entry.md"

record_sep=$(printf '\036')
unit_sep=$(printf '\037')

if [ -n "$log_range" ]; then
  git log --no-merges "--format=${record_sep}%h${unit_sep}%s${unit_sep}%b" "$log_range" > "$commits_file"
else
  : > "$commits_file"
fi

awk -v RS="$record_sep" -v FS="$unit_sep" -v version="$version" -v release_date="$release_date" '
BEGIN {
  order[1] = "Breaking Changes"
  order[2] = "Features"
  order[3] = "Fixes"
  order[4] = "Performance"
  order[5] = "Refactoring"
  order[6] = "Templates"
  order[7] = "Build and CI"
  order[8] = "Documentation"
  order[9] = "Tests"
  order[10] = "Maintenance"
  order[11] = "Other Changes"
  order_count = 11
}

function clean_subject(subject, cleaned) {
  cleaned = subject
  sub(/^[[:alpha:]][[:alnum:]_-]*(\([^)]*\))?!?:[[:space:]]*/, "", cleaned)
  return cleaned
}

function commit_type(subject, value) {
  value = subject
  if (value !~ /^[[:alpha:]][[:alnum:]_-]*(\([^)]*\))?!?:/) {
    return ""
  }
  sub(/(\([^)]*\))?!?:.*/, "", value)
  return value
}

function add_item(group, subject, hash, text, key) {
  text = clean_subject(subject)
  if (text == "") {
    text = subject
  }
  text = "- " text " (" hash ")"
  key = group SUBSEP text
  if (seen[key]++) {
    return
  }
  items[group, ++counts[group]] = text
}

NF >= 2 {
  hash = $1
  subject = $2
  body = $3
  if (hash == "" || subject == "") {
    next
  }

  type = commit_type(subject)
  breaking = subject ~ /^[[:alpha:]][[:alnum:]_-]*(\([^)]*\))?!:/ || body ~ /BREAKING CHANGE/

  if (breaking) {
    group = "Breaking Changes"
  } else if (type == "feat") {
    group = "Features"
  } else if (type == "fix") {
    group = "Fixes"
  } else if (type == "perf") {
    group = "Performance"
  } else if (type == "refactor") {
    group = "Refactoring"
  } else if (type == "template" || subject ~ /template|templates/) {
    group = "Templates"
  } else if (type == "build" || type == "ci") {
    group = "Build and CI"
  } else if (type == "docs") {
    group = "Documentation"
  } else if (type == "test") {
    group = "Tests"
  } else if (type == "chore" || type == "style" || type == "revert") {
    group = "Maintenance"
  } else {
    group = "Other Changes"
  }

  add_item(group, subject, hash)
}

END {
  print "## [" version "] - " release_date
  print ""
  printed = 0
  for (i = 1; i <= order_count; i++) {
    group = order[i]
    if (counts[group] > 0) {
      print "### " group
      print ""
      for (j = 1; j <= counts[group]; j++) {
        print items[group, j]
      }
      print ""
      printed = 1
    }
  }
  if (!printed) {
    print "### Changes"
    print ""
    print "- Release " version
    print ""
  }
}
' "$commits_file" > "$entry_file"

mkdir -p "$(dirname "$release_notes_file")"
{
  cat "$entry_file"
  echo "### Install"
  echo
  echo '```bash'
  echo "go install ${module_path}@${version}"
  echo '```'
} > "$notes_file"

if [ ! -f "$changelog_file" ]; then
  {
    echo "# Changelog"
    echo
    echo "All notable changes to this project are generated from Conventional Commits during release."
    echo
  } > "$changelog_file"
fi

awk -v version="$version" '
BEGIN { skip = 0 }
$0 ~ "^## \\[" version "\\]" {
  skip = 1
  next
}
skip && /^## \[/ {
  skip = 0
}
!skip {
  print
}
' "$changelog_file" > "$without_entry_file"

awk -v entry_file="$entry_file" '
function print_entry(  line) {
  while ((getline line < entry_file) > 0) {
    print line
  }
  close(entry_file)
}

BEGIN {
  inserted = 0
}

!inserted && /^## \[/ {
  print_entry()
  print ""
  inserted = 1
}

{
  print
}

END {
  if (!inserted) {
    if (NR > 0) {
      print ""
    }
    print_entry()
  }
}
' "$without_entry_file" > "$changelog_file"

cp "$notes_file" "$release_notes_file"

echo "Generated ${changelog_file} and ${release_notes_file} for ${version}."
