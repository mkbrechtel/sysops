<!--
SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# checker_network_checks

Orchestrator role for network connectivity checks.

## Requirements

- Debian 12/bookworm or 13/trixie

## Role Variables

None.

## Dependencies

- `mkbrechtel.sysops.checker_check_ping`

## Example Playbook

```yaml
- hosts: servers
  become: yes
  roles:
    - mkbrechtel.sysops.checker_network_checks
```
