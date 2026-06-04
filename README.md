# Orbital Package System (ops)

We package and deliver like UPS trucks.

## What

Every package system in existence is too hard to use and too painful to deal with. We want to fix that.
If you just want to ship a filesystem wrapped in a cryptographically verifiable archive you are in the right place.
Orbital is a reboot of [ZPS](https://zps.io) a project that was never fully realized.

## Why

- The usability of recent solutions is constrained by complexity
- Most other systems have co-mingled the transport functionality with the build system
- Everything else is mired in technical debt easily eliminated with modern advancements

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

## Installation

```sh
bash -c "$(curl -fsSL https://hub.platform.engineering/get/setup.sh)" -- install orbital
```

- Ensure that `/opt/pel/bin` is in your shell $PATH
- Add the following to `/opt/pel/.ops/tree.pkl` to work with other repositories FOR EXAMPLE:
```pkl
amends "orbital:/tree.pkl"

repositories = new Listing<Repository> {
  // Platform Engineering Labs tools repository  
  new {
    uri = "https://hub.platform.engineering/repos/platform.engineering/pel#stable"
  }

  // Formae hub community repository  
  new {
    uri = "https://hub.platform.engineering/repos/platform.engineering/community#stable"
  }
}
```

**Try it!** 

- run `ops refresh`
- run `ops list`
- run `ops install formae`

## License
Apache 2.0