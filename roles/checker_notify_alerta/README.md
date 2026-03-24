<!--
SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# checker_notify_alerta

Alerta alerting integration for sending check results to the Alerta monitoring platform.

## Requirements

- Debian 12/bookworm or 13/trixie

## Role Variables

See `defaults/main.yaml` for all available variables and their default values.

Key variables:

- `checker_notify_alerta_api_alert_url` - Alerta API endpoint
- `checker_notify_alerta_api_key` (required) - API key for Alerta
- `checker_notify_alerta_environment` (default: `'Development'`) - Environment tag for alerts

## Dependencies

- `mkbrechtel.sysops.checker`

## Example Playbook

```yaml
- hosts: servers
  become: yes
  roles:
    - role: mkbrechtel.sysops.checker_notify_alerta
      vars:
        checker_notify_alerta_api_key: your-api-key
        checker_notify_alerta_environment: Production
```
