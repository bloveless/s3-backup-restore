#!/usr/bin/env sh
set -e

AWS_ACCESS_KEY="${ACCESS_KEY}"
AWS_SECRET_KEY="${SECRET_KEY}"

if [ "$AWS_ACCESS_KEY" = "" ]; then
  echo "Using AWS_ACCESS_KEY from docker secrets."
  AWS_ACCESS_KEY=$(xargs < /run/secrets/AWS_ACCESS_KEY)
else
  echo "Using AWS_ACCESS_KEY from environment."
fi

if [ "$AWS_SECRET_KEY" = "" ]; then
  echo "Using AWS_SECRET_KEY from docker secrets."
  AWS_SECRET_KEY=$(xargs < /run/secrets/AWS_SECRET_KEY)
else
  echo "Using AWS_SECRET_KEY from environment."
fi

aws configure set aws_secret_access_key "$AWS_SECRET_KEY"
aws configure set aws_access_key_id "$AWS_ACCESS_KEY"
aws configure set default.region "$AWS_REGION"

exec "$@"
