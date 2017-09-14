ferryd
--------

[![Report](https://goreportcard.com/badge/github.com/solus-project/binman)](https://goreportcard.com/report/github.com/solus-project/ferryd) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

`ferryd` is the binary repository manager for Solus. This is a reimplementation of the current `binman` Python code, in keeping with Solus' goals of using `Go` for this kind of tooling.

`ferryd` aims to provide simple management of Solus repositories, with a git-like feel, and a rigid approach to caching, by pooling assets and hardlinking them into the appropriate place.


**Note**: `ferryd` is currently a work in progress, and is currently our highest priority work item.

ferryd is a [Solus project](https://solus-project.com/).

![logo](https://build.solus-project.com/logo.png)

Requirements
------------

`ferryd` should provide easy maintainence of `eopkg` repositories without the need for any host side tooling. It should provide git like syntax to enable creation, modification, and deletion of repositories.

It should support the concept of a processing queue, where we wait for a full upload payload to become available, before then merging them into the repository database.

To reduce the cost of indexing a repository (many thousands of packages) a simple database should be used to enable very quick dumping to the `eopkg-index.xml`.

`ferryd` should also support the automatic creation of `.delta.eopkg` delta packages to reduce the cost of update for users. While the existing `binman.py` implementation can do all these things, it is very limited, inefficient, and often misses delta opportunities.

TODO
----

 - [ ] Restore delta op per package
 - [ ] Restore delta operation for whole repo
 - [ ] Fire off delta job for **each** new package in the transit manifest - parallel
 - [ ] Get delta inclusion working
 - [ ] Mark failed deltas
 - [ ] Then have per-delta fire off sequential Index job for the entire repo (cheap enough)
 - [ ] Handle garbage collection of deltas when including new delta
 - [ ] Handle garbage collection of all deltas when removing a package
 - [ ] Add `clone` operation to clone one repo to another (optionally all or tip)
 - [ ] Add `pull` operation to pull from one repo into another (missing and mismatched)
 - [ ] Add delete operation to remove repo (unref cycle)
 - [ ] Add trim commands
 - [ ] Maybe add `trim` subcommand to nuke obsoletes ?
 - [ ] Throw another shedload of data and test upload cycle/bump upload/delta/index
 - [ ] Stats UI? i.e. ongoing jobs, recently completed, etc.
 - [ ] Restore binman parity, allow removing package (by release number), copying a single package, group of packages, etc.


Usage (basic)
-------------

Start ferryd to monitor `ferryd.sock`:

    ./bin/ferryd

Create a repo:

    ./bin/ferryctl create-repo testing

Add packages:

    ./bin/ferryctl import testing path/to/eopkgs

License
-------

Copyright © 2016-2017 Solus Project, ferryd Developers (See AUTHORS file)

`ferryd` is available under the terms of the Apache-2.0 license
