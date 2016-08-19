# Change Log
All notable changes to this project will be documented in this file.

## [Unreleased][unreleased]
### Fixed

### Added

### Removed

### Changed

## [v3.1] - 2016-05-23
### Fixed
- Panic when renaming stopped container #183
- won't start without route configuration #185
- RouteManager.Name() didn't return name
### Added
- update container name if we get a rename event. closes #144 (#180)

### Removed

### Changed
- Now using Alpine Linux 3.3 and GO 1.5.3, removed the "edge" package repo for building the official Docker image (#174)
- Fix exposed ports in Dockerfile and readme. Remove references to /tmp/docker.sock from readme

## [v3] - 2016-03-03
### Fixed
- use start/die like old version not create/destroy
- performance fix, generalizing SyslogMessage, minor cleanups
- Initialize Route options map
- Fixed a couple of typos, updated narrative
- UDP message delivery should not kill the program
- Exit with return code 1 on job setup failure
- Simplify and add early exit to RoutingFrom
- Unmarshal without buffering
- Remove unnecessary closure
- Undo change introduced in 07555c5
- Fix port number in httpstream example
- Use correct nilvalue for structured data as per rfc 5424
- retry tcp errors and don't hang forever on failure

### Added
- mention irc channel
- allowing easy custom builds of logspout
- Allow env vars in stream URLs
- Allow you to ignore log messages from individual containers by setting container environment variable, LOGSPOUT=ignore, when starting
- Add URL for Logstash module
- Adding CircleCI, Docker and IRC badges to readme.
- Add TLS transport. Fixes #116

### Removed
- Removed attach on restart event
- remove dev containers
- Removed deprecated library hosted in google code in favor of its new home

### Changed
- switched to gliderlabs org
- assume build
- rough pass at breaking logspout.go into separate packages
- fully split up packages. major refactoring of router
- simpler matching. working routesapi. dropped old utils
- make sure all uri params get into route options
- readme updates and module specific readmes
- renamed ConnectionFactory to AdapterTransport
- updated readme to use current schema
- names and parama
- more readable
- hold handler from returning until streamer finishes
- primarily designed new boot output, but came with it architectural changes
- updating docker sock location
- support old location for docker socket
- force link in case its run again, such as with custom builds
- analytics test
- update analytics
- Update README.md
- Update README with tls module
- Wrong port in README.md #136


## [v2] - 2015-02-12
### Added
- Allow comma-separated routes on boot
- Added project versioning
- Development Dockerfile and make task
- Deis sponsorship / support

### Removed
- Staging binary. Built entirely in Docker.
- Dropped unnecessary layers in Dockerfile

### Changed
- Base container is now Alpine
- Moved to gliderlabs organization

[unreleased]: https://github.com/gliderlabs/logspout/compare/v3.1...HEAD
[v3.1]: https://github.com/gliderlabs/logspout/compare/v3...v3.1
[v3]: https://github.com/gliderlabs/logspout/compare/v2...v3
[v2]: https://github.com/gliderlabs/logspout/compare/v1...v2
