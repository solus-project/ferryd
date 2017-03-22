binman
--------

[![Report](https://goreportcard.com/badge/github.com/solus-project/binman)](https://goreportcard.com/report/github.com/solus-project/binman) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

`binman` is the binary repository manager for Solus. This is a reimplementation of the current `binman` Python code, in keeping with Solus' goals of using `Go` for this kind of tooling.

`binman` aims to provide simple management of Solus repositories, with a git-like feel, and a rigid approach to caching, by pooling assets and hardlinking them into the appropriate place.


**Note**: `binman` is currently a work in progress, and isn't an **immediate** Solus tooling goal.

binman is a [Solus project](https://solus-project.com/).

![logo](https://build.solus-project.com/logo.png)

Requirements
------------

`binman` should provide easy maintainence of `eopkg` repositories without the need for any host side tooling. It should provide git like syntax to enable creation, modification, and deletion of repositories.

It should support the concept of a processing queue, where we wait for a full upload payload to become available, before then merging them into the repository database.

To reduce the cost of indexing a repository (many thousands of packages) a simple database should be used to enable very quick dumping to the `eopkg-index.xml`.

`binman` should also support the automatic creation of `.delta.eopkg` delta packages to reduce the cost of update for users. While the existing `binman.py` implementation can do all these things, it is very limited, inefficient, and often misses delta opportunities.

License
-------

Copyright © 2016-2017 Solus Project

`binman` is available under the terms of the Apache-2.0 license
