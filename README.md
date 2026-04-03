# Orbital Package System (ops)

We package and deliver like UPS trucks.

## What

Every package system in existence is too hard to use and too painful to deal with. We want to fix that.
If you just want to ship a filesystem wrapped in a cryptographically verifiable archive you are in the right place.
Orbital is a reboot of [ZPS](https://zps.io) a project that was never fully realized.

## Why

- NIX is too complex, build systems should not be tightly coupled to package systems
- Everything else was designed for 1970 and it shows

## Notable features

- Crypto
  - Signed archives
  - Signed metadata
  - Public key fetching from DNS

- Embeddable
    - Use it with no Pkl dependency to create a software updater for your Go app
  
- Opkg
  - Fast random access signed archive format
  - Zstd compression
  - Variable build system friendly DSL (Opkgfile)
  - Version time component because why increment a semver for CI builds

- Repositories
  - Publish to S3
  - Fetch from HTTPS/S3

## License
Apache 2.0