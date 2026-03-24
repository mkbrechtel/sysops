<!--
SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# checker_notify_email

Email notification integration for check results with rate limiting.

## Requirements

- Debian 12/bookworm or 13/trixie

## Role Variables

See `defaults/main.yaml` for all available variables and their default values.

Key variables:

- `checker_notify_email_enabled` (default: `true`) - Enable/disable email notifications
- `checker_notify_email_to` (default: `'root@localhost'`) - Recipient email address
- `checker_notify_email_rate_limit` (default: `10`) - Max emails per hour per check

## Dependencies

- `mkbrechtel.sysops.checker`

## Example Playbook

```yaml
- hosts: servers
  become: yes
  roles:
    - role: mkbrechtel.sysops.checker_notify_email
      vars:
        checker_notify_email_to: admin@example.com
```
