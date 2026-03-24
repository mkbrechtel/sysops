#!/bin/bash

# SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
#
# SPDX-License-Identifier: AGPL-3.0-or-later

# Ensure required variables are set
if [ -z "$1" ]; then
    echo "Error: Usage: $0 <run_file>" >&2
    exit 9
fi

RUN_FILE="$1"

# Run check and tee output, using PIPESTATUS to get check.sh's exit code
./check.sh 2>&1 | tee "${RUN_FILE}"
exit ${PIPESTATUS[0]}
