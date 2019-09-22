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

if [ "$RESTORE_DELETE" = "true" ]; then
  RESTORE_ARGS="$RESTORE_ARGS --delete"
fi

find_latest() {
  # find the latest in any of the folders hourly, daily, weekly, monthly
  BACKUPS=$(aws s3 ls --human-readable "$S3_PATH/hourly/" | grep -E '^.*PRE' | sed -e 's/.*PRE \(.*\)\//hourly\/\1/g')
  BACKUPS="$BACKUPS
$(aws s3 ls --human-readable "$S3_PATH/daily/" | grep -E '^.*PRE' | sed -e 's/.*PRE \(.*\)\//daily\/\1/g')"
  BACKUPS="$BACKUPS
$(aws s3 ls --human-readable "$S3_PATH/weekly/" | grep -E '^.*PRE' | sed -e 's/.*PRE \(.*\)\//weekly\/\1/g')"
  BACKUPS="$BACKUPS
$(aws s3 ls --human-readable "$S3_PATH/monthly/" | grep -E '^.*PRE' | sed -e 's/.*PRE \(.*\)\//monthly\/\1/g')"

  echo "$BACKUPS" | sort -r -t "/" -k 2 | head -n 1
}

clean() {
  # clean out the folder currently being worked on.
  echo "Cleaning backups"
  CLEAN_PATH="$S3_PATH/$1/"

  NUM_BACKUPS=0
  case "$OPERATION" in
    "hourly")
      NUM_BACKUPS=$((NUM_DAILY_BACKUPS + 1))
      ;;
    "daily")
      NUM_BACKUPS=$((NUM_HOURLY_BACKUPS + 1))
      ;;
    "weekly")
      NUM_BACKUPS=$((NUM_WEEKLY_BACKUPS + 1))
      ;;
    "monthly")
      NUM_BACKUPS=$((NUM_MONTHLY_BACKUPS + 1))
      ;;
  esac

  ALL_BACKUPS=$(aws s3 ls --human-readable "$CLEAN_PATH" | grep -E '^.*PRE' | sed -e 's/.*PRE \(.*\)\//\1/g' | sort -r)
  ALL_BACKUPS_COUNT=$(echo "$ALL_BACKUPS" | wc -l)

  echo "Currently found $ALL_BACKUPS_COUNT $OPERATION backups."
  if [ "$ALL_BACKUPS_COUNT" -ge "$NUM_BACKUPS" ]; then
    echo "Proceeding with delete of $((ALL_BACKUPS_COUNT-NUM_BACKUPS+1)) $OPERATION backups."
    BACKUPS_TO_REMOVE=$(echo "$ALL_BACKUPS" | tail -n "+$NUM_BACKUPS")
    echo "$BACKUPS_TO_REMOVE"
    echo "$BACKUPS_TO_REMOVE" | while read -r line; do
      eval aws s3 rm --recursive "$S3_PATH/$OPERATION/$line"
    done
  fi
}

restore() {
  if [ "$RESTORE_DATE" = "latest" ]; then
    echo "Finding latest backup in all folders (hourly/daily/weekly/monthly)"
    RESTORE_DATE=$(find_latest)
    echo "RESTORE_DATE: $RESTORE_DATE"
  fi

  if [ -n "$RESTORE_DATE" ]; then
    eval aws s3 sync "$S3_PATH/$RESTORE_DATE" "$DATA_DIRECTORY" "$RESTORE_ARGS"
    echo "Restored from backup: $RESTORE_DATE"

    if [ "$CHOWN_ENABLE" = "true" ]; then
      echo "Applying new owner ($CHOWN_UID) and group ($CHOWN_GID) to restored files"
      chown -R "$CHOWN_UID:$CHOWN_GID" "$DATA_DIRECTORY"
    fi
  else
    echo "No backup was found. Skipping restore"
  fi
}

backup() {
  if [ "$WRITE_BACKUP_DATE" = "true" ]; then
    echo "Creating BACKUP_DATE file"
    echo "$OPERATION/$CUR_DATE" > "$DATA_DIRECTORY/BACKUP_DATE"

    if [ "$CHOWN_ENABLE" = "true" ]; then
      echo "Applying new owner ($CHOWN_UID) and group ($CHOWN_GID) to BACKUP_DATE file"
      chown -R "$CHOWN_UID:$CHOWN_GID" "$DATA_DIRECTORY/BACKUP_DATE"
    fi
  fi

  echo "Uploading data directory ($DATA_DIRECTORY) to s3 path ($S3_PATH)"
  eval aws s3 sync "$DATA_DIRECTORY" "$S3_PATH/$1/$CUR_DATE" "$BACKUP_ARGS"

  echo "$OPERATION backup complete: $CUR_DATE"
}

if { [ "$OPERATION" = "restore" ] && [ "$DATA_FILES_COUNT" -eq 0 ]; } || [ "$RESTORE_FORCE" = "true" ]; then
  echo "No files found or restore forced, attempting restore from $RESTORE_DATE"
  restore
else
  echo "Found files in $DATA_DIRECTORY. And RESTORE_FORCE was not set to \"true\". Skipping restore"
fi

if [ "$OPERATION" = "hourly" ] || [ "$OPERATION" = "daily" ] \
  || [ "$OPERATION" = "weekly" ] || [ "$OPERATION" = "monthly" ];
then
  backup "$OPERATION"
  clean "$OPERATION"
fi
