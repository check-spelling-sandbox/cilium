#!/usr/bin/env bash

# check-logging-subsys-field.sh checks whether all logging entry instances
# created from DefaultLogger contain the LogSubsys field. This is required for
# proper labeling of error/warning Prometheus metric and helpful for debugging.
# If any entry which writes any message doesn't contain the 'subsys' field,
# Prometheus metric logging hook (`pkg/metrics/logging_hook.go`) is going to
# fail.

# Directories:
# - pkg/debugdetection
# - test/
# - vendor/
# - _build/
# are excluded, because instances of DefaultLogger in those modules have their
# specific usage which doesn't break the Prometheus logging hook.

set -eu

files_missing_log_sub_system=$(mktemp)
git ls-files '*.go' |
    perl -ne 'next if m<(?:^|/)(?:vendor|test|pkg/debugdetection)/>;print' |
    xargs -n1 perl -n -e 'print "$ARGV:$.:$_\n" if (/(?!.*LogSubsys)log[ ]*= logging\.DefaultLogger.*/);' > "$files_missing_log_sub_system"
if [ -s "$files_missing_log_sub_system" ]; then
    cat "$files_missing_log_sub_system"
    echo "Logging entry instances have to contain the LogSubsys field. Example of"
    echo "properly configured entry instance:"
    echo
    echo -e "\timport ("
    echo -e "\t\t\"github.com/cilium/cilium/pkg/logging\""
    echo -e "\t\t\"github.com/cilium/cilium/pkg/logging/logfields\""
    echo -e "\t)"
    echo
    echo -e "\tvar log = logging.DefaultLogger.WithField(logfields.LogSubsys, \"my-subsystem\")"
    echo
    exit 1
fi
