#!/usr/bin/env sh
set -e
. "./.env.sh"

BACKUP_ARGS="$BACKUP_ARGS"
CUR_DATE="$(date --utc +%Y-%m-%dT%H:%M:%S)Z"
DATA_DIRECTORY="${DATA_DIRECTORY:=/data}"
DATA_FILES_COUNT=$(find "$DATA_DIRECTORY" -mindepth 1 | wc -l )
OPERATION=$1
RESTORE_ARGS="$RESTORE_ARGS"
RESTORE_DATE="${RESTORE_DATE:=latest}"
S3_PATH=$(echo "$S3_PATH" | sed -e 's/^\(.*\)\/$/\1/g')
S3_FULL_PATH="$S3_PATH/$CUR_DATE"

if [ "$RESTORE_DELETE" = "true" ]; then
  RESTORE_ARGS="$RESTORE_ARGS --delete"
fi

restore() {
  if [ "$RESTORE_DATE" = "latest" ]; then
    RESTORE_DATE=$(aws s3 ls --human-readable "$S3_PATH/" | sed -e 's/.*PRE \(.*\)\//\1/g' | sort -r | head -n 1)
    echo "RESTORE_DATE: $RESTORE_DATE"
  fi

  eval aws s3 sync "$S3_PATH/$RESTORE_DATE" "$DATA_DIRECTORY" "$RESTORE_ARGS"

  echo "Restored from backup: $RESTORE_DATE"
}

backup() {
  echo "Uploding data directory ($DATA_DIRECTORY) to s3 path ($S3_PATH)"

  eval aws s3 sync "$DATA_DIRECTORY" "$S3_FULL_PATH" "$BACKUP_ARGS"

  echo "$OPERATION backup complete: $CUR_DATE"
}

if { [ "$OPERATION" = "restore" ] && [ "$DATA_FILES_COUNT" -eq 0 ]; } || [ "$RESTORE_FORCE" = "true" ]; then
  echo "No files found or restore forced, attempting restore from $RESTORE_DATE."
  restore
else
  echo "Found files in $DATA_DIRECTORY. And RESTORE_FORCE was not set to \"true\". Skipping restore."
fi

if [ "$OPERATION" = "hourly" ] || [ "$OPERATION" = "daily" ] \
  || [ "$OPERATION" = "weekly" ] || [ "$OPERATION" = "monthly" ];
then
  backup "$OPERATION"
fi
