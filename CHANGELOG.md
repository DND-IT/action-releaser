# Changelog

## [0.2.3](https://github.com/DND-IT/action-releaser/compare/v0.2.2...v0.2.3) (2026-04-17)


### Bug Fixes

* detect merged PRs via merged_at, not the boolean merged field ([441c23c](https://github.com/DND-IT/action-releaser/commit/441c23c6fb9c888e783da5a99afbfa92b0695162))

## [0.2.2](https://github.com/DND-IT/action-releaser/compare/v0.2.1...v0.2.2) (2026-04-17)


### Features

* detect release PR merges via GitHub API ([0328f3c](https://github.com/DND-IT/action-releaser/commit/0328f3cc99d640e4c96a543685cd65000f9737d9))

## [0.2.1](https://github.com/DND-IT/action-releaser/compare/v0.2.0...v0.2.1) (2026-04-14)


### Bug Fixes

* **test:** use 2x input size so truncated+suffix is always shorter than original ([bed71f9](https://github.com/DND-IT/action-releaser/commit/bed71f99d6c497f0177e553177a456aa9f64f94f))

## [0.2.0](https://github.com/DND-IT/action-releaser/compare/v0.1.10...v0.2.0) (2026-04-14)


### ⚠ BREAKING CHANGES

* version-strategy no longer accepts "date-rolling" or "numeric-rolling". Use "calver" (calendar versioning, YYYY.MM.DD[.N]) instead of "date-rolling". Numeric-rolling has no replacement.

### Features

* replace date-rolling/numeric-rolling with calver, drop numeric-rolling ([4a316f8](https://github.com/DND-IT/action-releaser/commit/4a316f8eaae2c82002704210f7581532e0e8e547))


### Bug Fixes

* **test:** correct truncate assertion to compare against input length not byte constant ([7ad9908](https://github.com/DND-IT/action-releaser/commit/7ad990825b7cef130b6d386a15362d4b0807c718))

## [0.1.10](https://github.com/DND-IT/action-releaser/compare/v0.1.9...v0.1.10) (2026-04-13)


### Features

* add release-mode, pr-created, and tag-in-pr-mode outputs ([17cd822](https://github.com/DND-IT/action-releaser/commit/17cd822f2c8b37bf1cf9bfca5e5be23556985e7b))
* split release-url semantics and add structured PR outputs ([3435695](https://github.com/DND-IT/action-releaser/commit/343569559642d251a10856e01409db86b106b093))


### Bug Fixes

* **releasepr:** force-push branch on update to avoid closing open PR ([74a2b4b](https://github.com/DND-IT/action-releaser/commit/74a2b4ba1ba707e9f8fb78d6233f26fd83970f35))

## [0.1.9](https://github.com/DND-IT/action-releaser/compare/v0.1.8...v0.1.9) (2026-04-13)


### Bug Fixes

* use --unreleased flag for git-cliff in PR release mode ([b0f0afb](https://github.com/DND-IT/action-releaser/commit/b0f0afbe0d8182f0a350f1237b489511e62ed875))

## [0.1.8](https://github.com/DND-IT/action-releaser/compare/v0.1.7...v0.1.8) (2026-04-09)


### Features

* pin action.yaml image to versioned GHCR tag ([#8](https://github.com/DND-IT/action-releaser/issues/8)) ([0df573f](https://github.com/DND-IT/action-releaser/commit/0df573f8907ba150270cf9c768bd31dc286d4207))

## [0.1.7](https://github.com/DND-IT/action-releaser/compare/v0.1.6...v0.1.7) (2026-04-07)


### Features

* support .release.yaml in addition to .release.yml ([fdf084b](https://github.com/DND-IT/action-releaser/commit/fdf084b887f671f69c7a11209ba3074e2b2e24dc))


### Bug Fixes

* change action.yaml tag-prefix default to empty string ([2d601ea](https://github.com/DND-IT/action-releaser/commit/2d601ea9fb943cf392ad92c41f10a0c7de899973))

## [0.1.6](https://github.com/DND-IT/action-releaser/compare/v0.1.5...v0.1.6) (2026-04-07)


### Bug Fixes

* change default TagPrefix from "v" to empty string ([340d4fa](https://github.com/DND-IT/action-releaser/commit/340d4fa550511a46bb04e7065df5ba17aec3a951))

## [0.1.5](https://github.com/DND-IT/action-releaser/compare/v0.1.4...v0.1.5) (2026-03-31)


### Bug Fixes

* remove --include-path from git-cliff --bumped-version ([fd4bed6](https://github.com/DND-IT/action-releaser/commit/fd4bed6bdc151b629710f264db97da82271a8207))

## [0.1.4](https://github.com/DND-IT/action-releaser/compare/v0.1.3...v0.1.4) (2026-03-31)


### Bug Fixes

* use stable release branch names to prevent noisy PR diffs ([f1d3f46](https://github.com/DND-IT/action-releaser/commit/f1d3f46d5adf15928461c4e2d6149f175ae2d76b))

## [0.1.3](https://github.com/DND-IT/action-releaser/compare/v0.1.2...v0.1.3) (2026-03-31)


### Features

* add include-path input for monorepo commit scoping ([e568c4b](https://github.com/DND-IT/action-releaser/commit/e568c4b04c441b7d0faa0ec7a5a9950cfe601ac5))

## [0.1.2](https://github.com/DND-IT/action-releaser/compare/v0.1.1...v0.1.2) (2026-03-31)


### Bug Fixes

* strict tag-prefix matching to prevent monorepo cross-contamination ([dd7da33](https://github.com/DND-IT/action-releaser/commit/dd7da3313fbfcfc9987ae106551cd6ffdc892a2e))

## [0.1.1](https://github.com/DND-IT/action-releaser/compare/v0.1.0...v0.1.1) (2026-03-27)


### Features

* add release PR mode for gated releases ([bff03a0](https://github.com/DND-IT/action-releaser/commit/bff03a0f14ae88bedd835f748f8e0721aca0c71e))
* initial implementation of action-releaser ([6009e9f](https://github.com/DND-IT/action-releaser/commit/6009e9f7ba6b0442ba5b9ac5b00bacbe63706386))


### Bug Fixes

* add safe.directory for Docker container workspace trust ([7d6ada2](https://github.com/DND-IT/action-releaser/commit/7d6ada270cbf9e4e69c3e980e3a62d516cde4125))
* configure git auth for tag push in Docker containers ([ccad4d6](https://github.com/DND-IT/action-releaser/commit/ccad4d6b4d34b64efe47ad351d42dc7c2af4763c))
* read Docker action inputs with hyphens (INPUT_DRY-RUN not INPUT_DRY_RUN) ([c651d57](https://github.com/DND-IT/action-releaser/commit/c651d57629147ae71204bc47ba57274ca72c5c2e))
* use Dockerfile for action image until first GHCR publish ([c4e9eed](https://github.com/DND-IT/action-releaser/commit/c4e9eed774e9a725b0ca51d675558f47ad11dfc3))
* use token-embedded remote URL for git push auth ([4c189e2](https://github.com/DND-IT/action-releaser/commit/4c189e23892a1b76f9898f8edf38faaa0a130187))
