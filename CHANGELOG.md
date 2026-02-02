# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.1] - 2026-02-02

### Fixed
- Added missing README.md and meta/main.yml for motd role (Galaxy import requirement)

## [0.1.0] - 2026-02-02

### Changed
- Renamed collection from `mkbrechtel.sys` to `mkbrechtel.sysops`

### Added
- New `motd` role for message of the day configuration

### Fixed
- Fixed user SSH key configuration
- Fixed home directory mode permissions

## [0.0.3] - 2025-07-26

### Fixed
- Updated meta/runtime.yml with mandatory requires_ansible field set to '>=2.14.3'
- Fixed Galaxy import error about missing requires_ansible in meta/runtime.yml

## [0.0.2] - 2025-07-26

### Fixed
- Added missing README.md files for all roles (ansible, common, users)
- Added missing meta/main.yml files for all roles with proper galaxy_info
- Fixed Galaxy import errors by ensuring all roles have required documentation

## [0.0.1] - 2025-07-26

### Added
- Podman role for installing container runtime with DNS support
  - Installs podman, dnsmasq, containernetworking-plugins, and podman-compose
  - Includes dnsname and aardvark-dns plugins for container DNS resolution
  - Daemonless operation (no systemd service management)
- Comprehensive project documentation
  - Enhanced README.md with collection overview and usage examples
  - CHANGELOG.md following Keep a Changelog format
  - CODING.md with development guidelines
  - CLAUDE.md for AI assistant context
  - RELEASE.md with release process documentation
- GitHub Actions release workflow
  - Automatic collection build on version tags
  - Ansible Galaxy publishing support
- Existing roles from initial collection structure:
  - **ansible** role for Ansible configuration and tools setup
  - **common** role for base system configuration (packages, repos, locales, timezone, etc.)
  - **updates** role for system update management
  - **users** role for user account management with home directory configuration

[0.1.1]: https://github.com/mkbrechtel/sysops/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/mkbrechtel/sysops/compare/v0.0.3...v0.1.0
[0.0.3]: https://github.com/mkbrechtel/sysops/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/mkbrechtel/sysops/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/mkbrechtel/sysops/releases/tag/v0.0.1