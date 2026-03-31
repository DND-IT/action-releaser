# Changelog

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
