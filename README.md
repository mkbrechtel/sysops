# sysops Ansible Collection

The **sysops** collection provides base system configuration and management roles for Debian systems. This collection focuses on common system administration tasks, user management, and container runtime setup.

**⚠️ Development Phase Notice** 
*This collection is currently in development (version 0.x.x). Breaking changes may occur in any release until we reach version 1.0.0. APIs, role interfaces, and variable names are subject to change.*

## Installation

```bash
ansible-galaxy collection install mkbrechtel.sysops
```

## Requirements

- Ansible >= 2.14.3
- Debian 12/bookworm or 13/trixie

## Included Roles

- **ansible**: Ansible configuration and tools setup
- **common**: Base system configuration (packages, repos, locales, timezone, etc.)
- **updates**: System updates management
- **users**: User account management with home directory configuration
- **podman**: Podman container runtime with DNS support

## Usage

### Base System Setup

```yaml
- hosts: servers
  become: yes
  roles:
    - mkbrechtel.sys.common
    - mkbrechtel.sys.users
    - mkbrechtel.sys.podman
```

### User Management

```yaml
- hosts: servers
  become: yes
  roles:
    - role: mkbrechtel.sys.users
      vars:
        users:
          - name: alice
            groups: ['sudo', 'docker']
            shell: /bin/bash
```

## License

Apache-2.0
