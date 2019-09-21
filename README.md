# S3 Backup and Restore

## Description
Automatic backups to s3. Automatic restores from s3 if the volume is completely empty (configurable).
Allow forced restores which will overwrite everything in the mounted volume.
Force restores can be done either for a specific time or the latest.

## Goals
1. Accept credentials through docker secrets and env variables.
2. Backup to s3 on some cadence (configurable).
3. If the directory is empty then restore from the latest backup automatically.
4. Allow for disabling back and/or restore.

## Stretch
1. Rotated updates with configurable time periods.
  1. How many monthly backups would you like to keep?
  2. How many weekly backups would you like to keep?
  3. How many daily backups would you like to keep?
  4. How many hourly backups would you like to keep?
