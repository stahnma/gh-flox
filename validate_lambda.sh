#!/usr/bin/env bash


# This simply pulls down the latest json file from s3 to see the date. This is
# useful for validating the lambda function is working as expected.

s3cmd ls s3://flox-data-lake/gh-flox/* |  sort | tail -1
