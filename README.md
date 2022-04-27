# S3 Backup and Restore
A docker image for backing up and restoring data from s3.

**This image has been moved to bloveless/docker-images**

## Description
Automatic backups to s3. Automatic restores from s3 if the volume is completely empty (configurable).
Allow forced restores which will overwrite everything in the mounted volume.
Force restores can be done either for a specific time or the latest.

## Goals
- [x] Accept credentials through docker secrets and env variables.
- [x] Backup to s3 on some cadence (configurable).
- [x] If the directory is empty then restore from the latest backup automatically.
- [x] Separate sub-commands for backup, restore, and cron.
- [x] Clean up backups.

## Config
You'll need to pass in the AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY either using environment variables or docker
secrets. They should be found in the environment under the same names or `/run/secrets/AWS_ACCESS_KEY_ID` and
`/run/secrets/AWS_SECRET_ACCESS_KEY`.

AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, S3_BUCKET are the only required environment variables.

You can configure the backup and restore using the following environment variables:

`AWS_REGION`: The AWS region you are targeting. I.E. "us-west-2".

`AWS_ACCESS_KEY_ID`: The AWS access key with permission to write to your S3 bucket. This may be omitted if using docker
secrets.

`AWS_SECRET_ACCESS_KEY`: The AWS secret key with permission to write to your S3 bucket. This may be omitted if using
docker secrets.

`S3_BUCKET`: The S3 bucket you'd like to use as your backup/restore bucket.

`S3_PATH`: You may specify a path within your bucket to use for backing up and restoring. Defaults to "/".

`DIRECTORY_PERMISSIONS`: The permissions to use when creating new directories. Defaults to 0755. _Files will be
automatically restored to the same permissions they had when they were backed up._

`CADENCE_HOURLY`: Cron schedule for running hourly backups. Defaults to "0 * * * *".

`CADENCE_DAILY`: Cron schedule for running daily backups. Defaults to "10 1 * * *".

`CADENCE_WEEKLY`: Cron schedule for running weekly backups. Defaults to "10 2 * * 0".

`CADENCE_MONTHLY`: Cron schedule for running monthly backups. Defaults to "10 3 1 * *".

`CHOWN_ENABLE`: Enable changing the permissions of files during restore. If enabled the entire DATA_DIRECTORY will
have it's ownership changed including files that weren't contained in the backup file. Defaults to "false".

`CHOWN_UID`: The UID that will be used when changing the ownership of the files. Defaults to 1000.

`CHOWN_GID`: The GID that will be used when changing the group ownership of the files. Defaults to 1000.

`DATA_DIRECTORY`: The directory where the backup and restore will occur. Defaults to "/data".

`NUM_BACKUPS_HOURLY`: The number of hourly backups to keep. Can be disabled by setting to 0. Defaults to 3.

`NUM_BACKUPS_DAILY`: The number of daily backups to keep. Can be disabled by setting to 0. Defaults to 3.

`NUM_BACKUPS_WEEKLY`: The number of weekly backups to keep. Can be disabled by setting to 0. Defaults to 3.

`NUM_BACKUPS_MONTHLY`: The number of monthly backups to keep. Can be disabled by setting to 0. Defaults to 3.

`RESTORE_FILE`: Set this if you'd like to restore from a specific date. NOTE: This should exactly match the date folder
within the S3 bucket. I.E. "hourly/2019-09-21T19:35:32Z.tar.gz" the S3_PATH will be added automatically.

`RESTORE_FORCE`: You can force a restore by setting this to "true". This will ignore if the directory already has files
in it. Defaults to "false".

## Notes
Restore will update existing files but will not remove files that haven't been backed up before. You may request that
the folder is cleaned before a restore is performed.

## Cleanup
Backups are rotated with a configurable number of backups to keep.
  1. How many monthly backups would you like to keep? Defaults to 3.
  2. How many weekly backups would you like to keep? Defaults to 3.
  3. How many daily backups would you like to keep? Defaults to 3.
  4. How many hourly backups would you like to keep? Defaults to 3.

If you aren't going to be using hourly backups or daily backups you can disable those backups by setting the appropriate
number of backups to 0. I.E. NUM_HOURLY_BACKUPS = 0 and NUM_DAILY_BACKUPS = 0.

## Logging
All logs are written to /var/log/run.log

## Examples
I built this for use in kubernetes, but I'd imagine you could use this in any orchestrator. For my example I use an init
container which only does a restore if the /data directory is completely empty. Then, a backup container which is
responsible for running the backup cron and backing up the files periodically.

```yaml
      initContainers:
        - name: s3-restore
          image: bloveless/s3-backup-restore:1.0.2
          volumeMounts:
            - name: public-files
              mountPath: /data
          args: ["restore"]
          tty: true
          env:
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: backup-keys
                  key: aws_access_key_id
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: backup-keys
                  key: aws_secret_access_key
            - name: AWS_REGION
              value: "us-west-2"
            - name: S3_BUCKET
              value: "my-backup-bucket"
            - name: S3_PATH
              value: "my-backup-path"
            - name: CHOWN_ENABLE
              value: "true"
            - name: CHOWN_UID
              value: "1000"
            - name: CHOWN_GID
              value: "1000"
      containers:
        - name: s3-backup
          image: bloveless/s3-backup-restore:1.0.2
          volumeMounts:
            - name: public-files
              mountPath: /data
          args: ["cron"]
          env:
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: backup-keys
                  key: aws_access_key
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: backup-keys
                  key: aws_secret_key
            - name: AWS_REGION
              value: "us-west-2"
            - name: S3_BUCKET
              value: "my-backup-bucket"
            - name: S3_PATH
              value: "my-backup-path"
            - name: CHOWN_ENABLE
              value: "true"
            - name: CHOWN_UID
              value: "1000"
            - name: CHOWN_GID
              value: "1000"
```
