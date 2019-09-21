# S3 Backup and Restore
A quick docker image to backup and restore data with s3.

## Description
Automatic backups to s3. Automatic restores from s3 if the volume is completely empty (configurable).
Allow forced restores which will overwrite everything in the mounted volume.
Force restores can be done either for a specific time or the latest.

## Goals
-[x] Accept credentials through docker secrets and env variables.
-[ ] Backup to s3 on some cadence (configurable).
-[x] If the directory is empty then restore from the latest backup automatically.
-[x] Allow for disabling backup and/or restore.

## Config
You'll need to pass in the AWS_ACCESS_KEY and AWS_SECRET_KEY either using environment variables or docker secrets.
They should be found in the environment under the same names or `/run/secrets/AWS_ACCESS_KEY`
and `/run/secrets/AWS_SECRET_KEY`.

You can configure the backup and restore using the following environment variables:

`AWS_REGION`: The AWS region you are targeting. I.E. "us-west-2"

`DATA_DIRECTORY`: The directory where the volume is mounted and where the backup and restore will occur. By default
this is set to "/data".

`RESTORE_ARGS`: Any additional flags you'd like to pass to the aws sync command on restore. I.E. "--follow-symlinks"

`RESTORE_DATE`: Set this if you'd like to restore from a specific date. NOTE: This should exactly match the date folder
within the S3 bucket. I.E. "2019-09-21T19:35:32Z"

`RESTORE_DELETE`: If you like the aws sync command to remove any files locally that don't match your remote backup.
This adds the --delete flag to the aws sync command.
I.E. "true"

`RESTORE_FORCE`: If you'd like to force a restore set this to "true".

`S3_PATH`: The s3 bucket and folder you'd like to use. I.E. "s3://backup-bucket/example-app"

## Stretch
1. Rotated updates with configurable time periods.
  1. How many monthly backups would you like to keep?
  2. How many weekly backups would you like to keep?
  3. How many daily backups would you like to keep?
  4. How many hourly backups would you like to keep?
