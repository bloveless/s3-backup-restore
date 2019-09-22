#!/usr/bin/env sh
set -e

AWS_ACCESS_KEY="${AWS_ACCESS_KEY}"
AWS_SECRET_KEY="${AWS_SECRET_KEY}"
CADENCE_HOURLY=${CADENCE_HOURLY:="0 * * * *"}
CADENCE_DAILY=${CADENCE_DAILY:="10 1 * * *"}
CADENCE_WEEKLY=${CADENCE_WEEKLY:="10 2 * * 0"}
CADENCE_MONTHLY=${CADENCE_MONTHLY:="10 3 1 * *"}
CHOWN_ENABLE=${CHOWN_ENABLE:=false}
CHOWN_UID=${CHOWN_UID:=1000}
CHOWN_GID=${CHOWN_GID:=1000}
ENABLE_SCHEDULE=${ENABLE_SCHEDULE:=true}
NUM_HOURLY_BACKUPS=${NUM_HOURLY_BACKUPS:=3}
NUM_DAILY_BACKUPS=${NUM_DAILY_BACKUPS:=3}
NUM_WEEKLY_BACKUPS=${NUM_WEEKLY_BACKUPS:=3}
NUM_MONTHLY_BACKUPS=${NUM_MONTHLY_BACKUPS:=3}
WRITE_BACKUP_DATE=${WRITE_BACKUP_DATE:=true}

if [ "$AWS_ACCESS_KEY" = "" ]; then
  echo "Using AWS_ACCESS_KEY from docker secrets"
  AWS_ACCESS_KEY=$(xargs < /run/secrets/AWS_ACCESS_KEY)
else
  echo "Using AWS_ACCESS_KEY from environment"
fi

if [ "$AWS_SECRET_KEY" = "" ]; then
  echo "Using AWS_SECRET_KEY from docker secrets"
  AWS_SECRET_KEY=$(xargs < /run/secrets/AWS_SECRET_KEY)
else
  echo "Using AWS_SECRET_KEY from environment"
fi

aws configure set aws_secret_access_key "$AWS_SECRET_KEY"
aws configure set aws_access_key_id "$AWS_ACCESS_KEY"
aws configure set default.region "$AWS_REGION"

cat << EOF > /root/.env.sh
export AWS_REGION=$AWS_REGION
export BACKUP_ARGS=$BACKUP_ARGS
export CHOWN_ENABLE=$CHOWN_ENABLE
export CHOWN_UID=$CHOWN_UID
export CHOWN_GID=$CHOWN_GID
export DATA_DIRECTORY=$DATA_DIRECTORY
export NUM_HOURLY_BACKUPS=$NUM_HOURLY_BACKUPS
export NUM_DAILY_BACKUPS=$NUM_DAILY_BACKUPS
export NUM_WEEKLY_BACKUPS=$NUM_WEEKLY_BACKUPS
export NUM_MONTHLY_BACKUPS=$NUM_MONTHLY_BACKUPS
export RESTORE_ARGS=$RESTORE_ARGS
export RESTORE_DATE=$RESTORE_DATE
export RESTORE_DELETE=$RESTORE_DELETE
export RESTORE_FORCE=$RESTORE_FORCE
export S3_PATH=$S3_PATH
export WRITE_BACKUP_DATE=$WRITE_BACKUP_DATE
EOF

./run.sh restore

if [ "$ENABLE_SCHEDULE" = "true" ]; then
  echo "" | crontab -u root -

  if [ "$NUM_HOURLY_BACKUPS" -gt 0 ]; then
    (crontab -u root -l ; echo "$CADENCE_HOURLY /root/run.sh hourly >> /var/log/run.log 2>&1") | crontab -u root -
  fi

  if [ "$NUM_DAILY_BACKUPS" -gt 0 ]; then
    (crontab -u root -l ; echo "$CADENCE_DAILY /root/run.sh daily >> /var/log/run.log 2>&1") | crontab -u root -
  fi

  if [ "$NUM_WEEKLY_BACKUPS" -gt 0 ]; then
    (crontab -u root -l ; echo "$CADENCE_WEEKLY /root/run.sh weekly >> /var/log/run.log 2>&1") | crontab -u root -
  fi

  if [ "$NUM_MONTHLY_BACKUPS" -gt 0 ]; then
    (crontab -u root -l ; echo "$CADENCE_MONTHLY /root/run.sh monthly >> /var/log/run.log 2>&1") | crontab -u root -
  fi

  crond -f -l 8 -d 8 -L /dev/stdout
fi

