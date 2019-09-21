#!/usr/bin/env sh
set -e

CUR_DATE="$(date --utc +%Y-%m-%dT%H:%M:%S)Z"
DATA_DIRECTORY="${DATA_DIRECTORY:=/data}"
DATA_FILES_COUNT=$(find "$DATA_DIRECTORY" -mindepth 1 | wc -l )
S3_PATH=$(echo "$S3_PATH" | sed -e 's/^\(.*\)\/$/\1/g')
S3_FULL_PATH="$S3_PATH/$CUR_DATE"
RESTORE_DATE="${RESTORE_DATE:=latest}"
RESTORE_ARGS="$RESTORE_ARGS"
BACKUP_ARGS="$BACKUP_ARGS"

if [ "$RESTORE_DELETE" = "true" ]; then
  RESTORE_ARGS="$RESTORE_ARGS --delete"
fi

restore() {
  if [ "$RESTORE_DATE" = "latest" ]; then
    RESTORE_DATE=$(aws s3 ls --human-readable "$S3_PATH/" | sed -e 's/.*PRE \(.*\)\//\1/g' | sort -r | head -n 1)
    echo "RESTORE_DATE: $RESTORE_DATE"
  fi

  set -x
  eval aws s3 sync "$S3_PATH/$RESTORE_DATE" "$DATA_DIRECTORY" "$RESTORE_ARGS"
  set +x
}

backup() {
  echo "Uploding data directory ($DATA_DIRECTORY) to s3 path ($S3_PATH)"

  set -x
  eval aws s3 sync "$DATA_DIRECTORY" "$S3_FULL_PATH" "$BACKUP_ARGS"
  set +x
}

if [ "$DATA_FILES_COUNT" -eq 0 ] || [ "$RESTORE_FORCE" = "true" ]; then
  echo "No files found attempting restore from $RESTORE_DATE."
  restore
else
  echo "Found files in $DATA_DIRECTORY. And RESTORE_FORCE was not set to \"true\". Skipping restore."
fi

if [ "$BACKUP_SKIP" != "true" ]; then
  backup
fi
