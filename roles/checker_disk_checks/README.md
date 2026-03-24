<!--
SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# checker_disk_checks

Auto-discovers mountpoints and creates disk checks for each.

## Requirements

- Debian 12/bookworm or 13/trixie

## Role Variables

See `defaults/main.yaml` for all available variables and their default values.

Key variables:

- `checker_disk_checks_excluded_filesystems` - Filesystem types to exclude
- `checker_disk_checks_excluded_mounts` (default: `[]`) - Specific mounts to exclude
- `checker_disk_checks_minimum_size_mb` (default: `128`) - Minimum mount size to monitor

## Dependencies

- `mkbrechtel.sysops.checker_check_disk`

## Example Playbook

```yaml
- hosts: servers
  become: yes
  roles:
    - mkbrechtel.sysops.checker_disk_checks
```
