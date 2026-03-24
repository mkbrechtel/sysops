<!--
SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Monitoring Checks

This directory contains instance-specific monitoring checks.
Each instance has its own subdirectory containing:
- check.sh: The main check script

Each check script should return appropriate exit codes:
- 0: Success
- 1: Warning
- 2: Critical
- 3+: Unknown/Error
