<!--
SPDX-FileCopyrightText: 2016-2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# sysops Ansible Collection

The **sysops** collection automates sysops tasks and provides a solution for deployment, monitoring, and backups of Debian-based machines.

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
- **common**: Base system configuration orchestrator (includes all roles below)
- **debian_apt_sources**: Debian APT sources configuration (deb822 format)
- **tools**: Base tools and common packages
- **storage**: Storage and filesystem tools
- **firmware**: CPU and device firmware packages
- **root_user**: Root user account configuration
- **ssh_agent**: SSH and GPG agent systemd user service setup
- **hostname**: Hostname and /etc/hosts configuration
- **locales**: System locale generation and configuration
- **timezone**: System timezone configuration
- **keyboard**: Keyboard layout configuration
- **resolvconf**: Resolvconf DNS configuration
- **sysctl_tweaks**: System sysctl performance tweaks
- **microcode**: CPU microcode updates
- **bash_shell**: Bash shell configuration
- **fish_shell**: Fish shell installation and configuration
- **zsh_shell**: Zsh shell configuration
- **updates**: System updates management
- **users**: User account management with home directory configuration
- **podman**: Podman container runtime with DNS support
- **managed**: High-level orchestrator for a fully managed system
- **setup_check**: Checker monitoring framework setup
- **setup_deploy**: Deployment infrastructure setup
- **setup_notify**: Unified notification setup
- **check**: Base check instance role
- **check_disk**: Disk space check
- **check_ram**: RAM/memory check
- **check_ping**: Network connectivity check
- **check_systemd**: Systemd service health check
- **notify_alerta**: Alerta notification integration
- **notify_email**: Email notification integration
- **deploy**: Base deploy instance role
- **deploy_ansible_play**: Ansible playbook deployment
- **deploy_ansible_pull**: Ansible pull deployment
- **test_deploy_ohai**: Test deployment (success)
- **test_deploy_fail**: Test deployment (failure)
- **triggered_by_git_hook**: Git hook trigger for deployments

## Usage

### Base System Setup

```yaml
- hosts: servers
  become: yes
  roles:
    - mkbrechtel.sysops.common
    - mkbrechtel.sysops.users
    - mkbrechtel.sysops.podman
```

### User Management

```yaml
- hosts: servers
  become: yes
  roles:
    - role: mkbrechtel.sysops.users
      vars:
        users:
          - name: alice
            groups: ['sudo', 'docker']
            shell: /bin/bash
```

## License

AGPL-3.0-or-later
