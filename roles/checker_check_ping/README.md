<!--
SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# checker_check_ping

Network connectivity monitoring using Nagios check_ping plugin.

## Requirements

- Debian 12/bookworm or 13/trixie

## Role Variables

- `check_ping_hostname` (required) - Hostname or IP to ping
- `checker_check_ping_cmd` - Full ping command (default computed from hostname)

## Dependencies

- `mkbrechtel.sysops.checker_check`

## Example Playbook

```yaml
- hosts: servers
  become: yes
  roles:
    - role: mkbrechtel.sysops.checker_check_ping
      vars:
        check_ping_hostname: google.de
```
