# Changelog

All notable changes to this project are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow [SemVer](https://semver.org).

## [Unreleased]

### Added
- `weheat` CLI over `weheat-go`: `login` plus one command per API endpoint.
- Token stored in `$XDG_CONFIG_HOME/weheat/token.json`; rotated refresh tokens are written back.
