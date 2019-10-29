# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [v1.0.3] - 2019-10-28
### Changed
Truncate `host` labels when more than 63 characters.

This is a limit imposed by kubernetes and we're hitting this with the new
domain. This label is used to find the app `Ingress`, `Deployment` and
`Service`.

Tools now truncate this label so we need to take this in account when
unidling. For existing resources it shouldn't make any difference (as these
labels wouldn't be longer than 63 characters)


## [v1.0.2] - 2019-06-26
### Fixed
Fixed the Docker image build problem caused by the now deleted jsonpatch package.


## [v1.0.1] - 2019-06-26
### Changed
Use strategic merge patch instead of brittle JSONPatch format.
Strategic merge patch is more robust and less likely to raise an error on patch.

For example we had problems where patching was failing while trying to remove
a key from a map because the key was not there. Strategic merge patch would
just work as that's the final/desired state.


## [v1.0.0] - 2019-03-29
### Changed
- Leverage host labels to simplify retrieval of k8s resources for the app to
  unidle
- start to use replicas-when-unidled annotation


## [v0.2.3] - 2019-03-21
### Changed
Removed #/6 "progress" messages as confusing


## [v0.2.2] - 2019-03-21
### Changed
Tweaks to wording to avoid possible confusions and don't show errors we can handle to users.


## [v0.2.1] - 2019-03-14
### Changed
**Better handling of connection errors and wording tweaks**
- treat connection errors differently: e.g. don't show undefined message and
  don't close EventSource to allow recover
- tell the user that wait for the deployment could take minutes
- don't send low-level k8s error to user when Service.Patch() failed
- improved logging messages to be more accurate of what actually happened


## [v0.2.0] - 2019-03-13
### Fixed
**Fix for undefined error while unidiling**
- fix for JS error undefined that we think it may have been caused by the
  unidler restoring the service too early in the unidling process
- users get more updates on the process, not just the bad news
- fixed white space between last message and image
- more logging
- refactorings and cleanups

See PR for full diff

Issue: ministryofjustice/analytics-platform#95
PR: https://github.com/ministryofjustice/analytics-platform-go-unidler/pull/5


## [v0.1.0] - 2019-01-24
### Changed
**Redirect Service instead of Ingress**
Requests to idled apps will receive an HTML page with dynamically updated
status of the unidling process


## [v0.0.5] - 2019-01-24
### Changed
**Report unidling progress**
Requests to idled apps will receive an HTML page with dynamically updated
status of the unidling process.


## [v0.0.4] - 2018-10-10
### Changed
- Improved logic to determine Deployment to unidle to use `app`
  label. This makes the unidling more flexible and be used
  on more apps.
- Improved wording and consistency of log messages


## [v0.0.3] - 2018-10-10
### Changed
- Improved wording of response when unidling failed.


## [v0.0.2] - 2018-10-10
### Added
- Added healthcheck endpoint (`/healthz`)

### Changed
- Improved README


## [v0.0.1] - 2018-10-09
### Added
- First go-unidler release, rewrite of [unidler](https://github.com/ministryofjustice/analytics-platform-unidler)
