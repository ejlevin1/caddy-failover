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
