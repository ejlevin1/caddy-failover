# [2.0.0](https://github.com/ejlevin1/caddy-failover/compare/v1.12.1...v2.0.0) (2025-08-19)


### Bug Fixes

* update test/openapi.Caddyfile to use new two-directive syntax ([6611ff7](https://github.com/ejlevin1/caddy-failover/commit/6611ff7fa1c3542310fd2a74cfc4f61a79c3e265))


### Features

* implement two-directive API registration system to prevent path mismatches ([a8bdb18](https://github.com/ejlevin1/caddy-failover/commit/a8bdb1861f3eb5bf74cdf4538297e192347a35dd))


### BREAKING CHANGES

* Split single caddy_api_registrar directive into two separate directives

- caddy_api_registrar: Registration directive (pass-through) that registers APIs at their actual paths
- caddy_api_registrar_serve: Serving directive that serves OpenAPI specs and documentation UIs

Changes:
- Split handler.go into registration_handler.go and serving_handler.go for cleaner separation
- Added explicit path requirement in registration directive to prevent mismatches
- Updated registry to track API paths separately from global configs
- Fixed GetFormatter to return nil for unknown formats instead of defaulting
- Added comprehensive test coverage for both new handlers
- Updated OpenAPI formatter to support configurable server URLs

Benefits:
- Eliminates path mismatches between documentation and actual endpoints
- Clearer separation of concerns between registration and serving
- APIs are registered exactly where they're mounted in handle directives
- Better validation and error handling

Testing:
- All existing tests pass with no regressions
- New handlers have 85%+ test coverage
- Manual testing confirms OpenAPI spec correctly shows /caddy/failover/status
- Swagger UI successfully calls endpoints at correct paths

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>

## [1.12.1](https://github.com/ejlevin1/caddy-failover/compare/v1.12.0...v1.12.1) (2025-08-18)


### Bug Fixes

* resolve OpenAPI endpoint path issues and improve documentation ([25690fb](https://github.com/ejlevin1/caddy-failover/commit/25690fb859d7546f485b873ec40c51156640170a))

# [1.12.0](https://github.com/ejlevin1/caddy-failover/compare/v1.11.0...v1.12.0) (2025-08-18)


### Bug Fixes

* improve tool detection feedback in initialize.sh ([662177d](https://github.com/ejlevin1/caddy-failover/commit/662177d0e96a12a1f9a7996b7a8e0dee2d1ea873))
* script now checks all tools including yamllint ([9dce490](https://github.com/ejlevin1/caddy-failover/commit/9dce4900546e2f4a4c41aa25087c2279add469e6))


### Features

* add usage descriptions for all tools in initialize.sh ([e22c158](https://github.com/ejlevin1/caddy-failover/commit/e22c158a4f13a23e99e2b3aa9587a16640112d09))
* enable integration tests on main branch and add dev setup script ([0a8cd20](https://github.com/ejlevin1/caddy-failover/commit/0a8cd204d53e1d25e2ce25ee47220fb550325679))

# [1.11.0](https://github.com/ejlevin1/caddy-failover/compare/v1.10.0...v1.11.0) (2025-08-18)


### Features

* improve health check logging and add active upstream tracking ([02a9e09](https://github.com/ejlevin1/caddy-failover/commit/02a9e09646a7f56633c4b46a9b7533233e47fbe8))

# [1.10.0](https://github.com/ejlevin1/caddy-failover/compare/v1.9.0...v1.10.0) (2025-08-18)


### Features

* add comprehensive OpenAPI test harness with 34+ API tests ([f14a465](https://github.com/ejlevin1/caddy-failover/commit/f14a4651d98dde16578e2f14193ebdbe5a0c79e8))
* add OpenAPI test harness and project documentation ([8515c7c](https://github.com/ejlevin1/caddy-failover/commit/8515c7c84728430de8bd9bb63d7c16281bfcb10b))

# [1.9.0](https://github.com/ejlevin1/caddy-failover/compare/v1.8.0...v1.9.0) (2025-08-18)


### Bug Fixes

* ensure failover status endpoint always returns data ([f583894](https://github.com/ejlevin1/caddy-failover/commit/f583894c4090d9d0562d2f90b5baae2ee39e13f7))
* prevent codecov rate limits by optimizing upload strategy ([18b3ebc](https://github.com/ejlevin1/caddy-failover/commit/18b3ebca851c84eb1862fff9132a25b45f519471))
* restore testdata files for CI/CD validation ([2bee3bd](https://github.com/ejlevin1/caddy-failover/commit/2bee3bd0d87c15aa9e2d5c2916e8fb1233f48052))


### Features

* implement format-agnostic API registrar module with UI support ([3e32cd7](https://github.com/ejlevin1/caddy-failover/commit/3e32cd796f37f491aa89aa1c3060e59518ebc199))

# [1.8.0](https://github.com/ejlevin1/caddy-failover/compare/v1.7.0...v1.8.0) (2025-08-17)


### Features

* add Docker loaded images with additional Caddy plugins ([2cd2936](https://github.com/ejlevin1/caddy-failover/commit/2cd2936d3b4bd2dd8edef1e8dbdcc8c5cc07d455))

# [1.7.0](https://github.com/ejlevin1/caddy-failover/compare/v1.6.3...v1.7.0) (2025-08-17)


### Bug Fixes

* add validation for proxy registration to prevent nil pointer errors ([9b65b1a](https://github.com/ejlevin1/caddy-failover/commit/9b65b1a1082b7c72acbbf91d18316bcc79e0b18a))
* automate branch protection setup with PAT support ([3a666f0](https://github.com/ejlevin1/caddy-failover/commit/3a666f0490e1e93bdbd0a077130e21aa0c48b97c))
* correct GitHub Actions job dependency from test-plugin to test ([55c7260](https://github.com/ejlevin1/caddy-failover/commit/55c72605dd60ada614dda65cfaa7f2018d3f1cf1))
* handle branch protection in release workflow ([7e939c0](https://github.com/ejlevin1/caddy-failover/commit/7e939c03a073cd09bd52972962890cebdcea46f9))
* improve test runner script and fix integration test ([ce6a751](https://github.com/ejlevin1/caddy-failover/commit/ce6a751f099fc2bff70f60ac0cb6124d97a3f252))
* improve TestConcurrentHealthChecks reliability ([a7e7110](https://github.com/ejlevin1/caddy-failover/commit/a7e7110d1e78e30a6cc982996f522d73ea6387a5))
* make browser opening optional in test script for CI ([63cec7a](https://github.com/ejlevin1/caddy-failover/commit/63cec7ad918f2eb0b07979bb0d8bc2630a097830))
* make token name configurable and remove hardcoded values ([04470a5](https://github.com/ejlevin1/caddy-failover/commit/04470a5bacde2e31d4ea1d31b0388ac94539156a))
* remove Go 1.21 from test matrix to match go.mod requirement ([2bd8bcb](https://github.com/ejlevin1/caddy-failover/commit/2bd8bcb811a8d01311a39a774227707df6e1e8ea))
* resolve data race conditions in tests ([8da8f09](https://github.com/ejlevin1/caddy-failover/commit/8da8f09555ddd685156c5cc6c9fe0b2c7ee6cb31))


### Features

* add comprehensive test runner script with multiple test modes ([6d26ab8](https://github.com/ejlevin1/caddy-failover/commit/6d26ab8cb7711ad753d7d1e958fd5f834b97fa63))
* enhance branch protection script with CLI options and fixes ([628f024](https://github.com/ejlevin1/caddy-failover/commit/628f024d7e85399c22b318de138e1d51efc39d72))
* enhance test infrastructure with formatted coverage reports ([82c3554](https://github.com/ejlevin1/caddy-failover/commit/82c3554dad71a67a21702b168957b57ab9561e90))
* restructure tests with preferred Go testing approaches ([82c304e](https://github.com/ejlevin1/caddy-failover/commit/82c304e8d448f81a81fbfb13343c949b3001431c))


### Reverts

* remove proxy registration validation that broke tests ([c825a9b](https://github.com/ejlevin1/caddy-failover/commit/c825a9b32e69c7798a3f92c78a48e93afe889989))

## [1.6.3](https://github.com/ejlevin1/caddy-failover/compare/v1.6.2...v1.6.3) (2025-08-16)


### Bug Fixes

* redesign proxy registry to fix status endpoint issues ([5c0a005](https://github.com/ejlevin1/caddy-failover/commit/5c0a005ad85863c77525bd7d9489c2e11b0c894f))

## [1.6.2](https://github.com/ejlevin1/caddy-failover/compare/v1.6.1...v1.6.2) (2025-08-16)


### Bug Fixes

* update Docker documentation with semantic versioning examples ([5de2478](https://github.com/ejlevin1/caddy-failover/commit/5de24782d4d91e98c7ab054ff1b44d5b85ccd6d5))

## [1.6.1](https://github.com/ejlevin1/caddy-failover/compare/v1.6.0...v1.6.1) (2025-08-16)


### Bug Fixes

* enable Docker image builds on semantic releases ([b172065](https://github.com/ejlevin1/caddy-failover/commit/b172065d983831d5473000ef4275e1342d7fae21))
* enable Docker image builds on semantic releases and optimize PR workflow ([3a2b6ce](https://github.com/ejlevin1/caddy-failover/commit/3a2b6ce36dd699611e6631a72f6ff7c31d6f0fbf))

# [1.6.0](https://github.com/ejlevin1/caddy-failover/compare/v1.5.0...v1.6.0) (2025-08-15)


### Features

* add active upstream tracking and improve path display in failover_status ([ce10a5b](https://github.com/ejlevin1/caddy-failover/commit/ce10a5b28d64bedd7d4085a199e3bb721daead2b))

# [1.5.0](https://github.com/ejlevin1/caddy-failover/compare/v1.4.0...v1.5.0) (2025-08-15)


### Bug Fixes

* failover_status returning null when status_path not specified ([daa3263](https://github.com/ejlevin1/caddy-failover/commit/daa3263ff4ccf3eb7c01a34f84bd948d020ad682))
* update integration test to use local mock server ([598780f](https://github.com/ejlevin1/caddy-failover/commit/598780f8315adc38fff6f6fa093d63bd9ea28299))


### Features

* add failover warning logs and custom health check user agent ([54c6710](https://github.com/ejlevin1/caddy-failover/commit/54c671033a7896b8a4edf7ce81ba65a2cb4d0d7b))

# [1.4.0](https://github.com/ejlevin1/caddy-failover/compare/v1.3.0...v1.4.0) (2025-08-15)


### Bug Fixes

* improve Docker image publishing with semantic versioning ([b44619f](https://github.com/ejlevin1/caddy-failover/commit/b44619f9ee34ab305d1c7251f1fbef7c10a9e8df))
* resolve invalid Docker tag format in build workflow ([be1e773](https://github.com/ejlevin1/caddy-failover/commit/be1e773dc1d6408c016536a963738c186f4dfe65))


### Features

* embed git information into Docker images ([cd3a89c](https://github.com/ejlevin1/caddy-failover/commit/cd3a89c17a7d2b1e87f437fa0c1657cfe0fecae3))

# [1.3.0](https://github.com/ejlevin1/caddy-failover/compare/v1.2.0...v1.3.0) (2025-08-15)


### Features

* add environment variable expansion support ([ca1a80c](https://github.com/ejlevin1/caddy-failover/commit/ca1a80c31bd8b8132b9a9cd5b7ac02a6bfc40405))

# [1.2.0](https://github.com/ejlevin1/caddy-failover/compare/v1.1.0...v1.2.0) (2025-08-15)


### Bug Fixes

* add path base support and debug logging to fix integration tests ([e2738c7](https://github.com/ejlevin1/caddy-failover/commit/e2738c7314df99e5f777d9738a7bd25236c08028))


### Features

* add per-upstream health check support ([1d15670](https://github.com/ejlevin1/caddy-failover/commit/1d15670d8b2c4410cfbe216aa54893857bb76f89))
* add status API endpoint for monitoring failover proxies ([bb120f2](https://github.com/ejlevin1/caddy-failover/commit/bb120f2bf329b25ef8499bf0458bab258a490686))

# [1.1.0](https://github.com/ejlevin1/caddy-failover/compare/v1.0.3...v1.1.0) (2025-08-15)


### Features

* add comprehensive debug logging for upstream selection ([6c64cdf](https://github.com/ejlevin1/caddy-failover/commit/6c64cdf2f520084b4b605b4836494318a41704de))
* add path base support and dynamic X-Forwarded-Proto header ([e5af32c](https://github.com/ejlevin1/caddy-failover/commit/e5af32c46559ff5e30472302a965fa4995b7f237))

## [1.0.3](https://github.com/ejlevin1/caddy-failover/compare/v1.0.2...v1.0.3) (2025-08-15)


### Bug Fixes

* update integration test to work with main branch behavior ([81927f8](https://github.com/ejlevin1/caddy-failover/commit/81927f873375f0b5b9632d3665bd9904a6221357))

## [1.0.2](https://github.com/ejlevin1/caddy-failover/compare/v1.0.1...v1.0.2) (2025-08-15)


### Bug Fixes

* remove invalid Docker tag prefix configuration ([2c69ac6](https://github.com/ejlevin1/caddy-failover/commit/2c69ac6a5fa243225eaf8edf51bd1e28f028ef53))
* remove sentence-case requirement from commitlint ([395662c](https://github.com/ejlevin1/caddy-failover/commit/395662c6fe0e5106f9b8658847ed29455eb64726))

## [1.0.1](https://github.com/ejlevin1/caddy-failover/compare/v1.0.0...v1.0.1) (2025-08-15)


### Bug Fixes

* remove npm cache from release workflow ([feaa848](https://github.com/ejlevin1/caddy-failover/commit/feaa84856bddf839721810bbce363dd5eae8fe1e))
