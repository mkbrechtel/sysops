# motd

Configure the system Message of the Day (MOTD).

## Requirements

- Debian 12/bookworm or 13/trixie

## Role Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `motd` | No | - | The message of the day content. If not defined, the role does nothing. |

## Example Playbook

```yaml
- hosts: servers
  become: yes
  roles:
    - role: mkbrechtel.sysops.motd
      vars:
        motd: |
          Welcome to {{ inventory_hostname }}
          Managed by Ansible
```

## License

Apache-2.0
