# S3 Backup and Restore
A quick docker image to backup and restore data with s3.

## Description
Automatic backups to s3. Automatic restores from s3 if the volume is completely empty (configurable).
Allow forced restores which will overwrite everything in the mounted volume.
Force restores can be done either for a specific time or the latest.

## Goals
- [x] Accept credentials through docker secrets and env variables.
- [x] Backup to s3 on some cadence (configurable).
- [x] If the directory is empty then restore from the latest backup automatically.
- [x] Allow for disabling backup and/or restore.
- [x] Clean up backups.

## Config
You'll need to pass in the AWS_ACCESS_KEY and AWS_SECRET_KEY either using environment variables or docker secrets.
They should be found in the environment under the same names or `/run/secrets/AWS_ACCESS_KEY`
and `/run/secrets/AWS_SECRET_KEY`.

You can configure the backup and restore using the following environment variables:

`AWS_REGION`: The AWS region you are targeting. I.E. "us-west-2"

`AWS_ACCESS_KEY`: The AWS access key with permission to write to your S3 bucket. May be omitted if using docker secrets.

`AWS_SECRET_KEY`: The AWS secret key with permission to write to your S3 bucket. May be omitted if using docker secrets.

`BACKUP_ARGS`: Any additional flags you'd like to pass to the aws sync command on backup. I.E. "--follow-symlinks"

`CADENCE_HOURLY`: Cron schedule for running hourly backups. Defaults to "0 * * * *".

`CADENCE_DAILY`: Cron schedule for running daily backups. Defaults to "10 1 * * *".

`CADENCE_WEEKLY`: Cron schedule for running weekly backups. Defaults to "10 2 * * 0".

`CADENCE_MONTHLY`: Cron schedule for running monthly backups. Defaults to "10 3 1 * *".

`CHOWN_ENABLE`: Enable changing the permissions of files during backup and restore. During backup only the BACKUP_DATE
file will have it's owner modified. During restore, all restored files will have their owner modified. Defaults to
"false"

`CHOWN_UID`: The UID that will be used when changing the ownership of the files. Defaults to 1000.

`CHOWN_GID`: The GID that will be used when changing the group ownership of the files. Defaults to 1000.

`DATA_DIRECTORY`: The directory where the volume is mounted and where the backup and restore will occur. Defaults to
"/data".

`ENABLE_SCHEDULE`: If set to false it will disable the schedule and exit after the first restore attempt. Use this
setting for init containers or the container will never exit. Defaults to "true".

`NUM_HOURLY_BACKUPS`: The number of hourly backups to keep. Can be disabled by setting to 0. Defaults to 3.

`NUM_DAILY_BACKUPS`: The number of daily backups to keep. Can be disabled by setting to 0. Defaults to 3.

`NUM_WEEKLY_BACKUPS`: The number of weekly backups to keep. Can be disabled by setting to 0. Defaults to 3.

`NUM_MONTLY_BACKUPS`: The number of montly backups to keep. Can be disabled by setting to 0. Defaults to 3.

`RESTORE_ARGS`: Any additional flags you'd like to pass to the aws sync command on restore. I.E. "--follow-symlinks"

`RESTORE_DATE`: Set this if you'd like to restore from a specific date. NOTE: This should exactly match the date folder
within the S3 bucket. I.E. "hourly/2019-09-21T19:35:32Z"

`RESTORE_DELETE`: If you like the aws sync command to remove any files locally that don't match your remote backup.
This adds the --delete flag to the aws sync command. Defaults to "false".

`RESTORE_FORCE`: If you'd like to force a restore set this to "true". Defaults to "false".

`S3_PATH`: The s3 bucket and folder you'd like to use. Make sure that if you are backing up multiple apps that you
choose a unique path for each one. Otherwise, you'll clean out backups from another application and restoring will not
be able to determine which app is being restored. I.E. "s3://backup-bucket/example-app"

`WRITE_BACKUP_DATE`: If set to "true" a file called BACKUP_DATE will be written to the root of the data directory every
time the folder is backed up. This may help in identifying the last time the data directory was backed up or when
verifying that a restore has taken place. Defaults to "true".

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
container which only does a restore. Then, a backup container which is responsible for running the backup cron and
backing up the files periodically.

```yaml
      initContainers:
        - name: s3-restore
          image: bloveless/s3-backup-restore:0.1.3-alpine
          volumeMounts:
            - name: public-files
              mountPath: /data
          env:
            - name: AWS_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: backup-keys
                  key: aws_access_key
            - name: AWS_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: backup-keys
                  key: aws_secret_key
            - name: AWS_REGION
              value: us-west-2
            - name: CHOWN_ENABLE
              value: "true"
            - name: CHOWN_UID
              value: "83"
            - name: CHOWN_GID
              value: "2003"
            - name: ENABLE_SCHEDULE
              value: "false"
            - name: S3_PATH
              value: s3://<BUCKET>/<PATH>
```

Note that the above snippet shows that the schedule is disabled (ENABLE_SCHEDULE="false") since this init container will
be used only for restoring data before the container which uses the public-files volume mount.

```yaml
      containers:
        - name: s3-backup
          image: bloveless/s3-backup-restore:0.1.3-alpine
          volumeMounts:
            - name: public-files
              mountPath: /data
          env:
            - name: AWS_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: backup-keys
                  key: aws_access_key
            - name: AWS_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: backup-keys
                  key: aws_secret_key
            - name: AWS_REGION
              value: us-west-2
            - name: CHOWN_ENABLE
              value: "true"
            - name: CHOWN_UID
              value: "83"
            - name: CHOWN_GID
              value: "2003"
            - name: S3_PATH
              value: s3://<BUCKET>/<PATH>
```

This container runs as a sidecar to the working container. Note that this one doesn't modify the ENABLE_SCHEDULE
variable since this container will be responsible for keeping S3 up to date with backups.
