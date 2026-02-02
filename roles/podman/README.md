# Podman Role

This role installs and configures Podman container runtime with DNS support on Debian/Ubuntu systems.

## Requirements

- Ansible >= 2.14
- Debian (bookworm, bullseye) or Ubuntu (jammy, focal)
- Root/sudo privileges for package installation

## Role Variables

Currently no variables are defined for this role.

## Dependencies

None.

## Example Playbook

```yaml
- hosts: servers
  become: yes
  roles:
    - mkbrechtel.sysops.podman
```

## What Gets Installed

- `podman` - Container runtime
- `dnsmasq` - DNS services
- `containernetworking-plugins` - Container networking support
- `podman-compose` - Docker Compose compatibility
- `golang-github-containernetworking-plugin-dnsname` - DNS name resolution plugin for containers
- `aardvark-dns` - DNS server for container name resolution

## License

Apache-2.0