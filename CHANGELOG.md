# Changelog

All notable changes to DevRites are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and DevRites adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Releases are generated automatically by [semantic-release](https://semantic-release.gitbook.io/) from Conventional Commits on `main`.

## [4.0.0](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.28...v4.0.0) (2026-08-03)

### ⚠ BREAKING CHANGES

* **repo:** use native host orchestration

### Changed

* **repo:** use native host orchestration ([0f871b2](https://github.com/ViktorsBaikers/DevRites/commit/0f871b2e2680290ed707006c693a91453619e079))

### Fixed

* **docs:** synchronize native orchestration contracts ([d80aaed](https://github.com/ViktorsBaikers/DevRites/commit/d80aaedf83d598b201060f51f0c58f570a7d676f))
* **tests:** admit LF index fixture on Windows ([1d3cc3b](https://github.com/ViktorsBaikers/DevRites/commit/1d3cc3bc693f35ea0b1f4d2ce0eeb0f3ba3006a0))
* **tests:** make Git fixtures portable on Windows ([287bcf6](https://github.com/ViktorsBaikers/DevRites/commit/287bcf65d15ef4a2a56913b358bee2ea22dff0b1))

## [3.2.28](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.27...v3.2.28) (2026-07-29)

### Fixed

* **agents:** bind durable V2 agent starts ([1da2cef](https://github.com/ViktorsBaikers/DevRites/commit/1da2cefd2a1e6b2ecb6a626224cc94fa0fc3662e))

## [3.2.27](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.26...v3.2.27) (2026-07-29)

### Fixed

* **rite:** separate recovery causes from symptoms ([c01f516](https://github.com/ViktorsBaikers/DevRites/commit/c01f5164ce349e5caad6ce450cfa6d76c7ce522b))

## [3.2.26](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.25...v3.2.26) (2026-07-29)

### Fixed

* **agents:** preserve hidden V2 agent types ([255591c](https://github.com/ViktorsBaikers/DevRites/commit/255591cff472193a4b87774912e6a6a8d9511968))

## [3.2.25](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.24...v3.2.25) (2026-07-28)

### Fixed

* **devrites:** close review safety gaps ([178a43e](https://github.com/ViktorsBaikers/DevRites/commit/178a43e5a25a7b0825c0e16aa76c402e6b48a3aa))

## [3.2.24](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.23...v3.2.24) (2026-07-28)

### Fixed

* **devrites:** close review safety gaps ([33e736a](https://github.com/ViktorsBaikers/DevRites/commit/33e736a3d9d69fd166827235ed1ce2310aeca75a))

## [3.2.23](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.22...v3.2.23) (2026-07-28)

### Fixed

* **devrites:** add retained-window abort recovery ([e47957c](https://github.com/ViktorsBaikers/DevRites/commit/e47957cd759f83e0166985b4b691d84068cc1d93))

## [3.2.22](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.21...v3.2.22) (2026-07-28)

### Fixed

* **devrites:** prevent receipt-only agent spawns ([4cf4c19](https://github.com/ViktorsBaikers/DevRites/commit/4cf4c19cf8671b8fdd497b8875c50bd50faf85d9))

## [3.2.21](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.20...v3.2.21) (2026-07-28)

### Fixed

* **devrites:** bound Codex dispatch stop retries ([4ef9d5b](https://github.com/ViktorsBaikers/DevRites/commit/4ef9d5baf3ca74b1d494f535e8820447e9c37129))

## [3.2.20](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.19...v3.2.20) (2026-07-28)

### Fixed

* **devrites:** preserve Engram identifiers ([ad68eff](https://github.com/ViktorsBaikers/DevRites/commit/ad68eff3a5925ebc7204ed250d5609ed5e41c515))

## [3.2.19](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.18...v3.2.19) (2026-07-28)

### Fixed

* **devrites:** recover ignored Codex reads ([9b0db60](https://github.com/ViktorsBaikers/DevRites/commit/9b0db603b298a8c8d7e36aafa61b909d11c00361))

## [3.2.18](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.17...v3.2.18) (2026-07-28)

### Fixed

* **devrites:** fail closed on filesystem errors ([e0e6c6c](https://github.com/ViktorsBaikers/DevRites/commit/e0e6c6c5c9114fc423cc704b8031d4bcb3728831))
* **devrites:** validate archive entry types ([02235fb](https://github.com/ViktorsBaikers/DevRites/commit/02235fb078ff4c15bdd36e58c57cc32e6bba8b75))

## [3.2.17](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.16...v3.2.17) (2026-07-27)

### Fixed

* **devrites:** harden agent orchestration and proof gates ([0602313](https://github.com/ViktorsBaikers/DevRites/commit/06023130b918117ae00ab0dadcfab6c000443981))
* **devrites:** normalize restore prefix on Windows ([27c5646](https://github.com/ViktorsBaikers/DevRites/commit/27c5646a1ed344e38a488dd6fc6584b67f1de09e))
* **devrites:** preserve restore bytes on Windows ([e6c3233](https://github.com/ViktorsBaikers/DevRites/commit/e6c32332e6ea9137644a8a6f42b542240b44c7e1))

## [3.2.16](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.15...v3.2.16) (2026-07-27)

### Fixed

* **rite:** run root-owned artifact gates ([7614022](https://github.com/ViktorsBaikers/DevRites/commit/7614022b36cf8c887f148e4d12777e20ee2ea3d4))

## [3.2.15](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.14...v3.2.15) (2026-07-27)

### Fixed

* **agents:** verify hidden V2 child roles ([e80b62b](https://github.com/ViktorsBaikers/DevRites/commit/e80b62bf53cec440cfad54a3d23d8d3cf5b120f3))

## [3.2.14](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.13...v3.2.14) (2026-07-26)

### Fixed

* **agents:** track unarmed retained retries ([44cf3cd](https://github.com/ViktorsBaikers/DevRites/commit/44cf3cd2d111060c76c53095ae76615ddba29c53))

## [3.2.13](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.12...v3.2.13) (2026-07-26)

### Fixed

* **agents:** retain canonical boundary before spawn ([b290aa9](https://github.com/ViktorsBaikers/DevRites/commit/b290aa97851915afcddd85ff43855fc8e07b6860))

## [3.2.12](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.11...v3.2.12) (2026-07-26)

### Fixed

* **agents:** capture wright boundary before spawn ([4eb95f3](https://github.com/ViktorsBaikers/DevRites/commit/4eb95f3f75c150b9c535972db6d84fadf97f1a87))

## [3.2.11](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.10...v3.2.11) (2026-07-26)

### Fixed

* **devrites:** bind canonical state at wright start ([b58a531](https://github.com/ViktorsBaikers/DevRites/commit/b58a531749b94f734b7abf7469656ad543a0cfa0))

## [3.2.10](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.9...v3.2.10) (2026-07-26)

### Fixed

* **devrites:** allow external dispatch scratch ([fdaac3f](https://github.com/ViktorsBaikers/DevRites/commit/fdaac3f9aaf19af897c6d3d70ce01fc046a60d3e))

## [3.2.9](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.8...v3.2.9) (2026-07-26)

### Fixed

* **devrites:** unblock retained reconcile checks ([c1093b0](https://github.com/ViktorsBaikers/DevRites/commit/c1093b00c3e32ba3cc22e0b9a2da548618ead621))

## [3.2.8](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.7...v3.2.8) (2026-07-26)

### Fixed

* **devrites:** prevent reconcile self-invalidation ([161ed18](https://github.com/ViktorsBaikers/DevRites/commit/161ed187e600dfc92623b22db5cd7f0824408d1a))

## [3.2.7](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.6...v3.2.7) (2026-07-26)

### Fixed

* **agents:** enforce conditional v2 dispatch ([f4169d4](https://github.com/ViktorsBaikers/DevRites/commit/f4169d41c456f352172be8d96afdcdc69800d849))
* **ci:** satisfy staticcheck error style ([0023265](https://github.com/ViktorsBaikers/DevRites/commit/00232656283647db8d94b2ed61dccb1629091b25))

## [3.2.6](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.5...v3.2.6) (2026-07-25)

### Changed

* **devrites:** remove package-existence gate ([4cbcbe9](https://github.com/ViktorsBaikers/DevRites/commit/4cbcbe9002eaf97f8f921e13e97b4dcfc6b61b34))

## [3.2.5](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.4...v3.2.5) (2026-07-25)

### Fixed

* **agents:** retain v2 wright dispatch receipt ([84f0c05](https://github.com/ViktorsBaikers/DevRites/commit/84f0c05c9d91d85194dfd4c97593ca1c07183282))

## [3.2.4](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.3...v3.2.4) (2026-07-25)

### Fixed

* **agents:** handle hidden v2 agent_type schema ([31ece9a](https://github.com/ViktorsBaikers/DevRites/commit/31ece9aad686fa3a3fa0308d5a3ede11f5b58c07))

## [3.2.3](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.2...v3.2.3) (2026-07-25)

### Fixed

* **agents:** enforce native specialist dispatch ([4ac3eb1](https://github.com/ViktorsBaikers/DevRites/commit/4ac3eb1a407ed8df4d3097984fbc0e369a5918b4))
* **agents:** satisfy staticcheck error style ([cc21c46](https://github.com/ViktorsBaikers/DevRites/commit/cc21c4600171111dc30065327a9cd0ffff2420bc))
* **skills:** avoid false invocation reference ([45f25bb](https://github.com/ViktorsBaikers/DevRites/commit/45f25bb70516460d753bccaa5ba78cce73e2a1d0))

## [3.2.2](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.1...v3.2.2) (2026-07-25)

### Fixed

* **agents:** restore guarded codex dispatch ([2d05e20](https://github.com/ViktorsBaikers/DevRites/commit/2d05e20f6196774827d1496275f4c3cb5c5e22e2))
* **ci:** govern bundled brace expansion advisory ([d1d0786](https://github.com/ViktorsBaikers/DevRites/commit/d1d078667fa42b61d8bec823136c628ca53fc6d8))

## [3.2.1](https://github.com/ViktorsBaikers/DevRites/compare/v3.2.0...v3.2.1) (2026-07-24)

### Fixed

* **ci:** govern temporary npm audit exceptions ([c2219e0](https://github.com/ViktorsBaikers/DevRites/commit/c2219e0ff11f11ee3f2e2ffa7c5d35b480c5eb76))
* **rite:** repair doctor compatibility ([811f339](https://github.com/ViktorsBaikers/DevRites/commit/811f339d22d3ec079549a5d50838d7596c72c7da))

## [3.2.0](https://github.com/ViktorsBaikers/DevRites/compare/v3.1.0...v3.2.0) (2026-07-24)

### Added

* **rite:** add semantic workspace upgrades ([3a7e51d](https://github.com/ViktorsBaikers/DevRites/commit/3a7e51d478fe9ed19c4de1ee4988e43d63ecf9a2))

## [3.1.0](https://github.com/ViktorsBaikers/DevRites/compare/v3.0.7...v3.1.0) (2026-07-24)

### Added

* **devrites:** harden planning and agent orchestration ([50be03b](https://github.com/ViktorsBaikers/DevRites/commit/50be03b90f1d5004ea010cf68ff07b74bab491c6))

### Fixed

* **ci:** correct invocation and eval discovery gates ([0a0d7d2](https://github.com/ViktorsBaikers/DevRites/commit/0a0d7d27546b90a7f5a2dd72bc9326f29416cc54))
* **devrites:** clear security and Windows mode gates ([59fcfc6](https://github.com/ViktorsBaikers/DevRites/commit/59fcfc6c1806975e36a69cde4e8a2c504657161a))
* **devrites:** restore static and Windows engine gates ([8aeb479](https://github.com/ViktorsBaikers/DevRites/commit/8aeb4793adf0ce57ba4dba7ddcb7e56f9f612f40))

## [3.0.7](https://github.com/ViktorsBaikers/DevRites/compare/v3.0.6...v3.0.7) (2026-07-22)

### Fixed

* **installer:** harden binary acquisition ([7534198](https://github.com/ViktorsBaikers/DevRites/commit/7534198dd3f6a29f065fe715e763e02bdc8a71c0))

## [3.0.6](https://github.com/ViktorsBaikers/DevRites/compare/v3.0.5...v3.0.6) (2026-07-22)

### Fixed

* **ci:** reject invalid workflow names ([8535414](https://github.com/ViktorsBaikers/DevRites/commit/853541455fcc79e28ca4d7044024da219683d700))
* **devrites:** align compliance matrix wording ([0bb1f28](https://github.com/ViktorsBaikers/DevRites/commit/0bb1f287c1ae2874a73669c1c35b081083f389af))
* **devrites:** isolate reconcile snapshot objects ([c02b1f1](https://github.com/ViktorsBaikers/DevRites/commit/c02b1f19a4717454f9e0194bf015fcc170dd37fc))
* **docs:** repair rite-ship standard link ([49db43c](https://github.com/ViktorsBaikers/DevRites/commit/49db43cdd6b42f15c7a7aaec3c47bcf7e9ceaa8d))

### Documentation

* **repo:** humanize repository documentation ([a5de735](https://github.com/ViktorsBaikers/DevRites/commit/a5de735db9cd3958a544df9802c6d1c7981470e8))
* **repo:** refresh README banner ([45b48c9](https://github.com/ViktorsBaikers/DevRites/commit/45b48c92f181e7c37b616993c2a98606b8984ee1))
* **repo:** streamline README guide ([932588f](https://github.com/ViktorsBaikers/DevRites/commit/932588f3fd1749d81e358f2be96873e59bdac8c1))

## [3.0.5](https://github.com/ViktorsBaikers/DevRites/compare/v3.0.4...v3.0.5) (2026-07-21)

### Fixed

* **deps:** update fast-uri ([1e14101](https://github.com/ViktorsBaikers/DevRites/commit/1e14101284496277f56d4560d5a59d87297f7daf))
* **devrites:** normalize Git root centrally ([a89ec40](https://github.com/ViktorsBaikers/DevRites/commit/a89ec408b6133629db3547dcb7960f29bec6a723))
* **devrites:** normalize Git root path ([38dccf0](https://github.com/ViktorsBaikers/DevRites/commit/38dccf0bac0bdf58f5fdcf171cac231c296af171))
* **devrites:** resolve packages by source ([ae21d9b](https://github.com/ViktorsBaikers/DevRites/commit/ae21d9b50b0b631fb4352e5820f1176213614b78))
* **release:** reconnect v3.0.4 lineage ([7b6f853](https://github.com/ViktorsBaikers/DevRites/commit/7b6f853a73bcd3bc6ea72fd4b6b10c024c69a468))
* **skills:** format human gate decisions clearly ([0781d99](https://github.com/ViktorsBaikers/DevRites/commit/0781d99ce081f76f5e69237396e99e78e1b027f9))

### Documentation

* **repo:** sync current behavior ([9555fbb](https://github.com/ViktorsBaikers/DevRites/commit/9555fbbe26ce896a5c99723e46a8c04b7fba06f4))

## [3.0.4](https://github.com/ViktorsBaikers/DevRites/compare/v3.0.3...v3.0.4) (2026-07-21)

### Fixed

* **ci:** harden supply chain checks ([d902574](https://github.com/ViktorsBaikers/DevRites/commit/d902574a7d30168fcef94aa4a3dae7707191fe8d))
* **deps:** update fast-uri ([1e14101](https://github.com/ViktorsBaikers/DevRites/commit/1e14101284496277f56d4560d5a59d87297f7daf))
* **devrites:** normalize canonical workspace files ([e70221f](https://github.com/ViktorsBaikers/DevRites/commit/e70221f5dd9fbda0f3a73c500a493497bc023a91))
* **devrites:** normalize Git root centrally ([a89ec40](https://github.com/ViktorsBaikers/DevRites/commit/a89ec408b6133629db3547dcb7960f29bec6a723))
* **devrites:** normalize Git root path ([38dccf0](https://github.com/ViktorsBaikers/DevRites/commit/38dccf0bac0bdf58f5fdcf171cac231c296af171))
* **devrites:** resolve packages by source ([ae21d9b](https://github.com/ViktorsBaikers/DevRites/commit/ae21d9b50b0b631fb4352e5820f1176213614b78))
* **skills:** format human gate decisions clearly ([0781d99](https://github.com/ViktorsBaikers/DevRites/commit/0781d99ce081f76f5e69237396e99e78e1b027f9))

### Documentation

* **repo:** sync current behavior ([9555fbb](https://github.com/ViktorsBaikers/DevRites/commit/9555fbbe26ce896a5c99723e46a8c04b7fba06f4))

## [3.0.3](https://github.com/ViktorsBaikers/DevRites/compare/v3.0.2...v3.0.3) (2026-07-20)

### Fixed

* **rite:** centralize workflow schema ([58bd55e](https://github.com/ViktorsBaikers/DevRites/commit/58bd55e8472a98be1073bcfa5fe7593e7bf0c5d1))

## [3.0.2](https://github.com/ViktorsBaikers/DevRites/compare/v3.0.1...v3.0.2) (2026-07-20)

### Fixed

* **rite:** read canonical workspace cursor state ([e2fd60c](https://github.com/ViktorsBaikers/DevRites/commit/e2fd60c13aa53637e1af02a792f36f3efe32e1c9))

## [3.0.1](https://github.com/ViktorsBaikers/DevRites/compare/v3.0.0...v3.0.1) (2026-07-20)

### Fixed

* **rite:** run analysis after task definition ([6373227](https://github.com/ViktorsBaikers/DevRites/commit/6373227521b5e1e6b22bfe48e98799e709f5dc7a))

## [3.0.0](https://github.com/ViktorsBaikers/DevRites/compare/v2.6.1...v3.0.0) (2026-07-17)

### ⚠ BREAKING CHANGES

* **release:** DevRites v3 removes legacy shell helper runtime paths.

Constraint: Release version must become v3.0.0.
Rejected: Commit local agent instruction files | user excluded them.
Rejected: Reintroduce shell helper wrappers | engine subcommands are canonical.
Confidence: high
Scope-risk: broad
Directive: Keep runtime entry points routed through devrites-engine.
Tested: bash tests/cli-smoke.sh; bash tests/npx-pack-smoke.sh
Tested: bash tests/mcp-test.sh; bash tests/hooks-parity-test.sh
Tested: bash scripts/validate.sh; go test ./...
Not-tested: live model-backed Codex exec and custom-subagent smoke are opt-in.

### Added

* **devrites:** add adoption workflow support ([cfb3f9e](https://github.com/ViktorsBaikers/DevRites/commit/cfb3f9e6fcbb9dd7834c3a125e192708ba9cb409))
* **devrites:** add engine parity golden fixtures ([f120773](https://github.com/ViktorsBaikers/DevRites/commit/f120773ff8287348b42b2be05fd27bd4e6ea3aaf))
* **devrites:** add profile and dogfood skills ([1f04ab7](https://github.com/ViktorsBaikers/DevRites/commit/1f04ab747c9d4c48c15d6f11b07032b7afd5c46b))
* **devrites:** add reconcile and integrity checks ([b37b69a](https://github.com/ViktorsBaikers/DevRites/commit/b37b69a47025e5f7ace79bc18bc9a07ca8702e2b))
* **devrites:** add reviewer surface validation ([14fd7db](https://github.com/ViktorsBaikers/DevRites/commit/14fd7dbc2edc9146e8daf8603f4414a7f229d6ee))
* **devrites:** harden validation and migration tests ([d44b0c0](https://github.com/ViktorsBaikers/DevRites/commit/d44b0c0c1a55c6fd7ff767ce9aa6e112712f32aa))
* **devrites:** improve hooks and doctor tooling ([77cf0fa](https://github.com/ViktorsBaikers/DevRites/commit/77cf0faf9bdbb56d1860a2056e66517d72063aa4))
* **devrites:** persist workflow evidence signals ([cc269a1](https://github.com/ViktorsBaikers/DevRites/commit/cc269a102e534f275b870cf77f5d3b6d596c17b8))
* **devrites:** port reviewer-readonly + subagent-orient hooks ([b388790](https://github.com/ViktorsBaikers/DevRites/commit/b388790e4f76cd96561dbcfbd0ced5946a7e9b18))
* **devrites:** port spec-validate/check-acceptance to Go engine ([a1d2791](https://github.com/ViktorsBaikers/DevRites/commit/a1d2791b832d9dd42a47e3092a3d1e34fdb1c56c))
* **devrites:** refine install and runtime helpers ([dba1e99](https://github.com/ViktorsBaikers/DevRites/commit/dba1e9988dbde6eaa5101026944e1c137c93a1d7))
* **devrites:** refresh agents and skill references ([ba33d0b](https://github.com/ViktorsBaikers/DevRites/commit/ba33d0bf89384454530fe0f79a0ab21964379094))
* **devrites:** refresh code review and plan guidance ([f4fb255](https://github.com/ViktorsBaikers/DevRites/commit/f4fb2553b8bfae8f6fbf629f14b70e3a439af282))
* **devrites:** refresh evals and skill outputs ([ab9dc94](https://github.com/ViktorsBaikers/DevRites/commit/ab9dc9447b858bc9492ceffe4c4657ff79661b8e))
* **devrites:** refresh explain and customize docs ([d57bdff](https://github.com/ViktorsBaikers/DevRites/commit/d57bdffd832eab3c76b83f031a853ac520e4de95))
* **devrites:** refresh frontend guidance and proof docs ([d26bb09](https://github.com/ViktorsBaikers/DevRites/commit/d26bb09ed8ebd6a8f779738e32b3daff0da19bd7))
* **devrites:** refresh plan and source-driven guidance ([63fce26](https://github.com/ViktorsBaikers/DevRites/commit/63fce26d1776010006c176ac9fe57bd359f3bacf))
* **devrites:** refresh prose craft and detection ([e95b757](https://github.com/ViktorsBaikers/DevRites/commit/e95b757acbaa66db1ee7fc968d07eae779e7ada0))
* **devrites:** refresh skill docs and validation tooling ([2ad1317](https://github.com/ViktorsBaikers/DevRites/commit/2ad1317a11f94bcae0a5911d48c8cd7f80911c41))
* **devrites:** refresh ux shape and spec intake ([f547fe6](https://github.com/ViktorsBaikers/DevRites/commit/f547fe6b0a676d27b56022771ece324593e256a4))
* **devrites:** refresh workflow and skill surfaces ([8b8d065](https://github.com/ViktorsBaikers/DevRites/commit/8b8d065a1f052ddf209fbaa1ddf375bc95932ffc))
* **devrites:** refresh workflow docs and checks ([c6e481d](https://github.com/ViktorsBaikers/DevRites/commit/c6e481d9b24eda43039d66b908551e37eb15d9a4))
* **devrites:** tighten engine reply contracts ([19ec4b0](https://github.com/ViktorsBaikers/DevRites/commit/19ec4b00411bf72428ce7b84f2f178fe9ff8f89d))
* **devrites:** update CI workflow ([2e7214a](https://github.com/ViktorsBaikers/DevRites/commit/2e7214acd6f7a13e16352b0948608b3a0323c147))
* **devrites:** update docs and engine behavior ([8a2dd35](https://github.com/ViktorsBaikers/DevRites/commit/8a2dd3573af3afe9c7bbc76aae8fa17aebed031e))
* **devrites:** update docs and eval tooling ([e51f3b3](https://github.com/ViktorsBaikers/DevRites/commit/e51f3b34cad8efc9c2e0eda0dc57717eccea4d42))
* **devrites:** update docs and release checks ([1d09f35](https://github.com/ViktorsBaikers/DevRites/commit/1d09f359faec7b65fdfa5ed9bed94311cf26e698))
* **devrites:** update engine gating and commands ([878ed2c](https://github.com/ViktorsBaikers/DevRites/commit/878ed2c3666fcc7d87af4de4c1a7db03a4c78d07))
* **devrites:** update engine internals and validation ([b4d6d92](https://github.com/ViktorsBaikers/DevRites/commit/b4d6d92bc0b29085fa123f433c57230b0610497f))
* **devrites:** update engine state and skills ([732c97e](https://github.com/ViktorsBaikers/DevRites/commit/732c97e3b7b9e37443f3168e824a6d108b9085c4))
* **devrites:** update hooks and guidance surfaces ([6e79143](https://github.com/ViktorsBaikers/DevRites/commit/6e7914339283ad557849e568eeffb213c0bfb342))
* **devrites:** update rite workflow docs and engine ([c8ff41e](https://github.com/ViktorsBaikers/DevRites/commit/c8ff41eb67b0b94af0b7cea728092e534b1e812b))
* **devrites:** update spec and workspace docs ([1f2888a](https://github.com/ViktorsBaikers/DevRites/commit/1f2888ace17e854c212cf58b4f62a1cc42bf741f))
* **installer:** make host installs engine-owned ([c70491c](https://github.com/ViktorsBaikers/DevRites/commit/c70491cb8d856d3c3ef6b9d017766e6b710bb6fc))
* **release:** ship v3.0.0 engine control plane ([da6ca62](https://github.com/ViktorsBaikers/DevRites/commit/da6ca6239d0321d0ab9bf3e36a7ffe46a80b6db3))

### Changed

* **repo:** reduce duplicate maintenance surfaces ([0e645d4](https://github.com/ViktorsBaikers/DevRites/commit/0e645d45fdf4f75cbb367016ace88ae79c40870c))

### Fixed

* **ci:** generate artifacts before validation ([0d3afa3](https://github.com/ViktorsBaikers/DevRites/commit/0d3afa3d4dd72e2b31b31467e759e42becfb0c8a))
* **ci:** make engine tests cross-platform ([1631450](https://github.com/ViktorsBaikers/DevRites/commit/163145057575d9833b1c28d0d1f00516e025f363))
* **ci:** normalize remaining Windows output ([8da9a61](https://github.com/ViktorsBaikers/DevRites/commit/8da9a6106188655f612cd3be7f32374a4a25435f))
* **installer:** keep Claude hooks complete in existing settings ([d9f57af](https://github.com/ViktorsBaikers/DevRites/commit/d9f57afd2d7fc60a44e6141887b0b7c6a9a23a1d))
* **tests:** normalize Windows golden files ([7d0a07c](https://github.com/ViktorsBaikers/DevRites/commit/7d0a07c5c4838237572d4807b17a475f1fd1ed71))

### Documentation

* **skills:** keep skill guidance compact ([6afebfe](https://github.com/ViktorsBaikers/DevRites/commit/6afebfeccf152669c44e96a99cd6610200945dc6))

## [2.6.1](https://github.com/ViktorsBaikers/DevRites/compare/v2.6.0...v2.6.1) (2026-07-05)

### Fixed

* **installer:** emit valid Codex hooks config ([5a23d0a](https://github.com/ViktorsBaikers/DevRites/commit/5a23d0a0d4f1d990cf206f258a0b6866b46cf341))

## [2.6.0](https://github.com/ViktorsBaikers/DevRites/compare/v2.5.2...v2.6.0) (2026-07-05)

### Added

* **installer:** add Codex support ([cd19c70](https://github.com/ViktorsBaikers/DevRites/commit/cd19c70e6c6df5a38a3524c4cb3f8e9885ffedc3))

## [2.5.2](https://github.com/ViktorsBaikers/DevRites/compare/v2.5.1...v2.5.2) (2026-06-28)

### Changed

* **skills:** swap browser-harness for Playwright MCP ([9c13e72](https://github.com/ViktorsBaikers/DevRites/commit/9c13e72c94e81f7df6dc7d07bf23acdfd4d18ddd))

## [2.5.1](https://github.com/ViktorsBaikers/DevRites/compare/v2.5.0...v2.5.1) (2026-06-28)

### Fixed

* **rite:** ask HITL build gaps inline instead of deferring to resolve ([33266f1](https://github.com/ViktorsBaikers/DevRites/commit/33266f19d9ad30f53fc8ef267f486a5562be61c7))

## [2.5.0](https://github.com/ViktorsBaikers/DevRites/compare/v2.4.0...v2.5.0) (2026-06-28)

### Added

* **installer:** prune files dropped from the pack on update ([830352a](https://github.com/ViktorsBaikers/DevRites/commit/830352ae19b2726cadd627ab7ca4f12b1b85c7fc))

### Fixed

* **devrites:** close skill-pack SSOT drift + consolidate review dispatch ([649db40](https://github.com/ViktorsBaikers/DevRites/commit/649db4040a120d04c35e91b8f69d51f610df9a5f))

## [2.4.0](https://github.com/ViktorsBaikers/DevRites/compare/v2.3.0...v2.4.0) (2026-06-28)

### Added

* **devrites:** enforce the doubt gate from invocation through seal ([6ff29d0](https://github.com/ViktorsBaikers/DevRites/commit/6ff29d0caa3384d6ba85f480369d0fb01d80fad1))

### Fixed

* **devrites:** calibrate doubt-coverage and close per-slice skip ([5bc09eb](https://github.com/ViktorsBaikers/DevRites/commit/5bc09eb94062f728bd855a2a581784e4e1197522))

## [2.3.0](https://github.com/ViktorsBaikers/DevRites/compare/v2.2.0...v2.3.0) (2026-06-23)

### Added

* **agents:** auto-firing devex-reviewer + retrospector ([425c582](https://github.com/ViktorsBaikers/DevRites/commit/425c582802b5efb18a463a31174420dfb96838fe))
* **rules:** add project invariants as a trusted gating layer ([e569d7e](https://github.com/ViktorsBaikers/DevRites/commit/e569d7e065d1baf4e974c274d921431cc3c6e26e))
* **skills:** add spec-grammar validator + scenario hooks ([765430c](https://github.com/ViktorsBaikers/DevRites/commit/765430c3f21bd9f139d9de9ecc4d73e292970505))
* **skills:** competitive forge builds + structured visual verdict ([c509519](https://github.com/ViktorsBaikers/DevRites/commit/c50951957525a5cbf673ee52d3fa573838092959))
* **tests:** behavioral evals for gating-skill discipline ([9cfde94](https://github.com/ViktorsBaikers/DevRites/commit/9cfde942cd6fa12d2f08d0cb4f8e3c11354b9355))

### Fixed

* **scripts:** allow forge-report.md as a runtime workspace artifact ([e01bd50](https://github.com/ViktorsBaikers/DevRites/commit/e01bd50dcc2d5767d8abd3a1f527ec762dc83583))

### Documentation

* **skills:** sync skill/agent/rule counts + catalogue with current pack ([57ea677](https://github.com/ViktorsBaikers/DevRites/commit/57ea6771b2040382b80848cd5c866952b46355c9))

## [2.2.0](https://github.com/ViktorsBaikers/DevRites/compare/v2.1.0...v2.2.0) (2026-06-23)

### Added

* **rules:** add observability + deprecation rules, OWASP LLM Top 10 ([0c29760](https://github.com/ViktorsBaikers/DevRites/commit/0c29760e504c22c18d7ca2c5e1986746ba95a07d))

### Fixed

* **release:** drop duplicate 2.1.0 changelog block ([0a7b510](https://github.com/ViktorsBaikers/DevRites/commit/0a7b5101ecd05e4c0b4d3812cefc68c0f92f1626))

## [2.1.0](https://github.com/ViktorsBaikers/DevRites/compare/v2.0.0...v2.1.0) (2026-06-23)

### Added

* **agents:** code reviewer gets structural-depth lenses ([b37c9d2](https://github.com/ViktorsBaikers/DevRites/commit/b37c9d259b99613b971269fac2943977dedbea6d))
* **agents:** perf reviewer gets Source/Measured CWV modes ([be82b20](https://github.com/ViktorsBaikers/DevRites/commit/be82b20b37252a9f55d77e0f71659ec58a213ca6))

## [2.0.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.18.0...v2.0.0) (2026-06-23)

### ⚠ BREAKING CHANGES

* **installer:** drop the Claude Code plugin install path

### Added

* **installer:** add npx devrites full-pack installer + npm publishing ([316a0d3](https://github.com/ViktorsBaikers/DevRites/commit/316a0d372dad1afe2fade08f0045370fd5b3d801))

### Changed

* **release:** group changelog by Added/Changed/Removed ([ea5772a](https://github.com/ViktorsBaikers/DevRites/commit/ea5772abca3d44ee1da496f3d904bcbe3b9ccc21))

### Removed

* **installer:** drop the Claude Code plugin install path ([beb825f](https://github.com/ViktorsBaikers/DevRites/commit/beb825fcb0738d4b03b32dce056871c80616ab49))

## [1.18.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.17.0...v1.18.0) (2026-06-22)

### Features

* **skills:** add rite-frame goal-reframe + failure-mode self-audit ([ca181ee](https://github.com/ViktorsBaikers/DevRites/commit/ca181eee47c4c47c19ea8069cb650e38a7a26c87))

## [1.17.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.16.0...v1.17.0) (2026-06-22)

### Features

* **skills:** auto-refresh code-intelligence indexes after edits ([6cdfa6c](https://github.com/ViktorsBaikers/DevRites/commit/6cdfa6cc5291955f5c6362970bcbda15589bc050))

## [1.16.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.15.0...v1.16.0) (2026-06-21)

### Features

* **skills:** make code-intelligence tools optional and add context7 ([d8f95f7](https://github.com/ViktorsBaikers/DevRites/commit/d8f95f7333dd0c35fa75d4e190a05db86e82ba92))

## [1.15.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.14.0...v1.15.0) (2026-06-21)

### Features

* **skills:** make code-intelligence tools optional and add context7 ([d8f95f7](https://github.com/ViktorsBaikers/DevRites/commit/d8f95f7333dd0c35fa75d4e190a05db86e82ba92))
* **skills:** test-integrity gates, enforcement hooks, learning loop ([32d5102](https://github.com/ViktorsBaikers/DevRites/commit/32d5102f1e82313d939632a416a5df3b00a2a675))

### Bug Fixes

* **ci:** allowlist new runtime artifacts; mark defensive deny-list ([eec1440](https://github.com/ViktorsBaikers/DevRites/commit/eec14403a827d98283c0641ae1c71087b24fe311))

## [1.15.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.14.0...v1.15.0) (2026-06-21)

### Features

* **skills:** test-integrity gates, enforcement hooks, learning loop ([32d5102](https://github.com/ViktorsBaikers/DevRites/commit/32d5102f1e82313d939632a416a5df3b00a2a675))

### Bug Fixes

* **ci:** allowlist new runtime artifacts; mark defensive deny-list ([eec1440](https://github.com/ViktorsBaikers/DevRites/commit/eec14403a827d98283c0641ae1c71087b24fe311))

## [1.15.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.14.0...v1.15.0) (2026-06-21)

### Features

* **skills:** test-integrity gates, enforcement hooks, learning loop ([32d5102](https://github.com/ViktorsBaikers/DevRites/commit/32d5102f1e82313d939632a416a5df3b00a2a675))

## [1.14.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.13.0...v1.14.0) (2026-06-21)

### Features

* **skills:** add prose-craft skill and sharpen anti-slop code charter ([8a9e7e1](https://github.com/ViktorsBaikers/DevRites/commit/8a9e7e159487a53c89b46975a54f84ed875e409d))

## [1.13.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.12.0...v1.13.0) (2026-06-21)

### Features

* **skills:** add ui-taste enrichments + optional design-memory rollup ([316e7aa](https://github.com/ViktorsBaikers/DevRites/commit/316e7aa042d86ae173b7fa061437ced5af931b1f))

## [1.12.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.11.0...v1.12.0) (2026-06-21)

### Features

* **agents:** add prompt-injection-resistance baseline ([f260fc7](https://github.com/ViktorsBaikers/DevRites/commit/f260fc7b719c8049cb5ecd42b70649f17204080a)), closes [#14](https://github.com/ViktorsBaikers/DevRites/issues/14)
* **ci:** blocking pack security scan — injection + hidden unicode ([89a8838](https://github.com/ViktorsBaikers/DevRites/commit/89a88383150afd6803ba405198ffc732e1e8a323)), closes [#13](https://github.com/ViktorsBaikers/DevRites/issues/13)
* **ci:** supply-chain IOC scanner for npm lockfile ([ac9d079](https://github.com/ViktorsBaikers/DevRites/commit/ac9d07976707c45cb8503531b297f9049f314ea2)), closes [#16](https://github.com/ViktorsBaikers/DevRites/issues/16)
* **ci:** workflow-security validator + pin third-party actions ([4c060c9](https://github.com/ViktorsBaikers/DevRites/commit/4c060c9f3d293b4b43919e7583dfc1f058fab21e)), closes [#15](https://github.com/ViktorsBaikers/DevRites/issues/15)
* **skills:** add fan-out footprint at seal ([093b366](https://github.com/ViktorsBaikers/DevRites/commit/093b366b22f89a4e6851237c1aa458638a5aa910)), closes [#19](https://github.com/ViktorsBaikers/DevRites/issues/19)
* **skills:** add rite-adopt brownfield on-ramp skill ([c86b770](https://github.com/ViktorsBaikers/DevRites/commit/c86b7702bf4c8c77736689d369b49375e677a927)), closes [#21](https://github.com/ViktorsBaikers/DevRites/issues/21)
* **skills:** add rite-doctor health check (two-tier) ([71d2c7b](https://github.com/ViktorsBaikers/DevRites/commit/71d2c7b99fb48cc97de4dfeddf9469af521f180f)), closes [#18](https://github.com/ViktorsBaikers/DevRites/issues/18)
* **skills:** conventions ledger with write-at-seal ([5de1464](https://github.com/ViktorsBaikers/DevRites/commit/5de1464a5a1d1a0a453aee84d5a515d616a929db)), closes [#17](https://github.com/ViktorsBaikers/DevRites/issues/17)
* **skills:** ledger read-at-orient with fresh-observation-wins ([2abe806](https://github.com/ViktorsBaikers/DevRites/commit/2abe806f69cdbae78bd90b2431e6009fd6121aa0)), closes [#20](https://github.com/ViktorsBaikers/DevRites/issues/20)

## [1.11.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.10.0...v1.11.0) (2026-06-19)

### Features

* **skills:** add A1 pre-block hook, observe by default ([b7e76d5](https://github.com/ViktorsBaikers/DevRites/commit/b7e76d58f214076a7bd860b7fc3588c2476f5ed5))

## [1.10.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.9.0...v1.10.0) (2026-06-19)

### Features

* **installer:** auto-approve read-only scripts + orient hooks ([3807883](https://github.com/ViktorsBaikers/DevRites/commit/38078838d2f4eeb9303c35c42812bbcb9bd9f5fc))
* **skills:** gate orchestrator out of source edits (A1) ([c64a4a0](https://github.com/ViktorsBaikers/DevRites/commit/c64a4a0f2722089fa938b5291de64eca9e4648c7))

### Bug Fixes

* **skills:** resolve skills-audit findings across pack, evals, and CI ([1499ce2](https://github.com/ViktorsBaikers/DevRites/commit/1499ce274f714544f4983c8e6fc6a11ebe36ee25))
* **tests:** tolerate preserved settings.json in uninstall smoke ([1d1b3d5](https://github.com/ViktorsBaikers/DevRites/commit/1d1b3d56a0b348d8181fa9fae55e70b4f9e51fd9))

### Refactors

* **skills:** fold attempts, drop lanes note, signpost quick ([02180b5](https://github.com/ViktorsBaikers/DevRites/commit/02180b52dfa9254597b3ae0040690f556f40f5f2))

## [1.9.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.8.0...v1.9.0) (2026-06-19)

### Features

* **skills:** enforce test completeness + assertion strength ([4f825a1](https://github.com/ViktorsBaikers/DevRites/commit/4f825a1d1956138a6db778f8b54d7dcafb3876a8))
* **skills:** present gaps as ranked option sets with inline resolve ([db70a21](https://github.com/ViktorsBaikers/DevRites/commit/db70a21511ad3ebc17b32e32afed8b02dcd47a22))
* **skills:** research-driven workflow improvements ([83cb2e0](https://github.com/ViktorsBaikers/DevRites/commit/83cb2e00fb95cdbf974440f098e397ae7c9cde76))

## [1.8.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.7.0...v1.8.0) (2026-06-19)

### Features

* **skills:** add progress footer to all rite-* lifecycle commands ([6b5eda0](https://github.com/ViktorsBaikers/DevRites/commit/6b5eda051ddc86c0a4d67a42d147cf4e9b7fbd17))

## [1.7.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.6.0...v1.7.0) (2026-06-18)

### Features

* **scripts:** add outcome grader and MCP server ([6b7263c](https://github.com/ViktorsBaikers/DevRites/commit/6b7263cf220508dcb567a7db6f79f619ea185249))
* **skills:** add state gates, devrites CLI, and Gotchas sections ([25eaa49](https://github.com/ViktorsBaikers/DevRites/commit/25eaa49e55fce2b22734e8c977dd3a59db73c0d1))

### Documentation

* **docs:** sync README, CONTRIBUTING, SECURITY with changes ([cc89594](https://github.com/ViktorsBaikers/DevRites/commit/cc895942a2a12bcff056ca44c957bf39349f00ca))

## [1.6.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.5.1...v1.6.0) (2026-06-18)

### Features

* **skills:** add shared devrites-lib orientation preamble ([fd8c8fc](https://github.com/ViktorsBaikers/DevRites/commit/fd8c8fc97fb879c3d81190152ad6f31d971a3bde))

### Documentation

* **skills:** document the orientation preamble and devrites-lib ([4cd0d65](https://github.com/ViktorsBaikers/DevRites/commit/4cd0d6575f69a4b20dae6c5ffe01b9e4c232f93a))

## [1.5.1](https://github.com/ViktorsBaikers/DevRites/compare/v1.5.0...v1.5.1) (2026-06-17)

### Refactors

* **skills:** vet every plan at scaled depth, never skip ([ba61859](https://github.com/ViktorsBaikers/DevRites/commit/ba61859dc84871ebe9951dd1dfc68037e58fcb4f))

## [1.5.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.4.0...v1.5.0) (2026-06-17)

### Features

* **skills:** add /rite-vet engineering plan review before build ([8c22b6a](https://github.com/ViktorsBaikers/DevRites/commit/8c22b6af4558540f70853e2d56688f49d528dffa))

## [1.4.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.3.0...v1.4.0) (2026-06-17)

### Features

* **skills:** add /rite-temper strategic spec review ([aa95d4d](https://github.com/ViktorsBaikers/DevRites/commit/aa95d4d27735c654a5bea8daf31fb6bc2dfa1767))

## [1.3.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.2.0...v1.3.0) (2026-06-17)

### Features

* **agents:** add devrites-slice-wright write-capable executor ([66a7c14](https://github.com/ViktorsBaikers/DevRites/commit/66a7c1418b727da181f8294e3bf6fdd632fa5711))

## [1.2.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.1.1...v1.2.0) (2026-06-17)

### Features

* **skills:** plan UX/UI before code via devrites-ux-shape ([e4a45f2](https://github.com/ViktorsBaikers/DevRites/commit/e4a45f27346dc4ab22c5fb7acf215105e947e0a7))

## [1.1.1](https://github.com/ViktorsBaikers/DevRites/compare/v1.1.0...v1.1.1) (2026-06-17)

### Bug Fixes

* **skills:** slice count is always derived, never user-forced ([07eb724](https://github.com/ViktorsBaikers/DevRites/commit/07eb724a785793dd3d38ab4889b202d22d4ca2ef))

### Documentation

* **docs:** /rite-ship in manifest descriptions + autocomplete example ([7d3dfce](https://github.com/ViktorsBaikers/DevRites/commit/7d3dfcee6a9ae99f6ac8d05a6573501c7c8713da))
* **docs:** list /rite-autocomplete in README table, fix /ship alias ([b5613fa](https://github.com/ViktorsBaikers/DevRites/commit/b5613fa922d2c51cc03149d840d316cc12770a76))

## [1.1.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.5...v1.1.0) (2026-06-17)

### Features

* **skills:** add /rite-autocomplete — unattended full lifecycle ([acad7e5](https://github.com/ViktorsBaikers/DevRites/commit/acad7e555c4f188665c9a7a5db4dd3dbb4264375))
* **skills:** add /rite-ship — execute the ship + close the task ([cc7db95](https://github.com/ViktorsBaikers/DevRites/commit/cc7db95f079de8da824187a482451a34aa02b852))
* **skills:** sharpen the rite-spec interview loop + coverage gate ([72994ea](https://github.com/ViktorsBaikers/DevRites/commit/72994eab298661b39724af955d5988f47035df03))

### Refactors

* **skills:** seal decides only; git ladder moves to /rite-ship ([80f0fbf](https://github.com/ViktorsBaikers/DevRites/commit/80f0fbfae0a77417283ed35efa4f3c8eae921aa7))

### Documentation

* **docs:** reflect ship/autocomplete + seal-decides split ([e9d4a7e](https://github.com/ViktorsBaikers/DevRites/commit/e9d4a7e64ed8cc4b8b0c742dea055e647d112c57))
* **repo:** fix stale skill count + phantom devrites-rules refs ([3793e1b](https://github.com/ViktorsBaikers/DevRites/commit/3793e1bdf2fecd0c21999c003f5ff09d5b48aef0))

## [1.0.5](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.4...v1.0.5) (2026-06-16)

### Bug Fixes

* **installer:** round-trip rules-only flags, cover update.sh ([0e82a76](https://github.com/ViktorsBaikers/DevRites/commit/0e82a76389ad9787c8f7a776fabffb01de6c49f8))
* **rules:** per-skill core load, dedupe table, validating-gate teeth ([007096e](https://github.com/ViktorsBaikers/DevRites/commit/007096e1f001ac80e1ca6e990afd134d4a8a80cd))
* **scripts:** correct gate tally, add AFK cap + qid scripts ([f2e46e5](https://github.com/ViktorsBaikers/DevRites/commit/f2e46e5155f7ac43dabf28f6db0a34bca84f0b2c))
* **skills:** workspace state, AFK budget, evidence + reviewer scope ([a8e9e75](https://github.com/ViktorsBaikers/DevRites/commit/a8e9e7567ae12b8a5cece961786c196323c1847b))

### Documentation

* **docs:** reconcile counts, fix phantom names and loading model ([f29241e](https://github.com/ViktorsBaikers/DevRites/commit/f29241e9aebd026afdd570d88bba500e9aee8b29))

## [1.0.4](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.3...v1.0.4) (2026-05-28)

### Bug Fixes

* **docs:** quote inside mermaid edge label broke flow.md diagram ([5112ba3](https://github.com/ViktorsBaikers/DevRites/commit/5112ba3be1cfb9f47195e98a5b7d50927662b64f))

## [1.0.3](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.2...v1.0.3) (2026-05-28)

### Bug Fixes

* **installer:** list agents as file array, validate manifest sync ([97d5004](https://github.com/ViktorsBaikers/DevRites/commit/97d50049cdaf4877272e8f555b92b87d1be26887))

### Documentation

* **docs:** bash install is recommended, plugin path is partial ([e729d18](https://github.com/ViktorsBaikers/DevRites/commit/e729d1851498ccda5be7496ffff72d88fd2b4ce7))

## [1.0.2](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.1...v1.0.2) (2026-05-28)

### Bug Fixes

* **installer:** plugin.json must use string repo and ./-prefixed paths ([cb50c01](https://github.com/ViktorsBaikers/DevRites/commit/cb50c01efcd45f813cc9fa7aeee7f25795bfd503))

## [1.0.1](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.0...v1.0.1) (2026-05-28)

### Bug Fixes

* **ci:** repair dependabot, sync README on release, use bot author ([d928765](https://github.com/ViktorsBaikers/DevRites/commit/d928765a965e4c49d1618cf07842e907f28deb0a))

## 1.0.0 (2026-05-28)

### Features

* **repo:** ship DevRites skills pack ([0915d40](https://github.com/ViktorsBaikers/DevRites/commit/0915d40f0c88e81dc9c122f5c755c7975957fdd4))

### Bug Fixes

* **ci:** bypass commitlint on semantic-release commits, tidy README ([0cf52a3](https://github.com/ViktorsBaikers/DevRites/commit/0cf52a3f27216ab5edb8197ec22628d03f2e5e31))
* **ci:** sync lockfile and reject multiline descriptions without PyYAML ([0efa85f](https://github.com/ViktorsBaikers/DevRites/commit/0efa85f052612b94926cf3382f88510059e5a8e8))
