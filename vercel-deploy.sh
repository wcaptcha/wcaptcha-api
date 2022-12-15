#!/bin/sh
export $(xargs <.env.production)
vercel -e S3_ENDPOINT=$S3_ENDPOINT -e S3_BUCKET=$S3_BUCKET -e S3_ACCESS_KEY=$S3_ACCESS_KEY -e S3_SECRET_KEY=$S3_SECRET_KEY --prod

