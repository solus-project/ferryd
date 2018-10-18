ferryd
--------

> Fast, safe and reliable transit for the delivery of software updates to users.


[![Report](https://goreportcard.com/badge/github.com/solus-project/binman)](https://goreportcard.com/report/github.com/solus-project/ferryd) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

ferryd is a [Solus project](https://getsol.us/).

![logo](https://build.getsol.us/logo.png)

`ferryd` is the binary repository manager for Solus. In addition to providing basic management for repositories, it is also an asynchronous job-based daemon, processing incoming package
uploads from authorised builder machines. `ferryd` attempts to optimise all operations ahead of time, by caching all metadata required for repository indexes.

The primary goal for `ferryd` is to provide a daemon that constantly monitors new uploads, and processes them as fast as possible. This ensures new packages are available almost immediately.
Complex, long running operations, are run in the background within a dedicated worker pool. This allows new packages to turn up in batch, and the delta packages to be produced lazily. Once
those delta packages are available, they're inserted into the main repository (and will appear in the index.)

The design of `ferryd` allows us to blit a repository index from the database to disk very fast (around 2-3s for a large repository). Special care is taken to only perform atomic updates to the
index - meaning no connectivity issues for clients with corrupt or partial indexes. The repository index should always be available, and all published packages should *permanently* be present.

`ferryd` takes special care to cache wherever possible, and uses a reference-counted package pool. All package files within each repository are hard-linked from the pool tree, allowing to
save disk space through enforced deduplication. As such, a package's ID (the basename of the file) must be unique to a ferryd instance. Putting it all together, this allows us to simply "ref"
a package into a repository from the pool, which is used for very rapid clone and pull operations.

`ferryd` is the replacement for the aging `binman.py` script previously used by Solus, and is designed to combat the design mistakes of that implementation. Emphasis is placed on speed, scaling,
and having packages immediately and permanently available. Less delays for developers, and rapid updates and sync deployment to users.

Lastly, `ferryd` aims to provide very simple sync abilities to help control deployment of packages to other repositories. An explicit design goal is to enable "Pulling" a repository into an
existing repository, which in turn publishes one channel to another. This is used in Solus to control sync-windows from unstable to stable, and is done as a single atomic operation.

_Disclaimer regarding the name: Solus has this weird obsession with all things nautical. Oh and birds._


Usage (basic)
-------------

Start ferryd to monitor `ferryd.sock`:

    ./bin/ferryd -d myRepoBase -s ./ferryd.sock

Create a repo:

    ./bin/ferryctl -s ./ferryd.sock create-repo testing

Add packages:

    ./bin/ferryctl -s ./ferryd.sock import testing path/to/eopkgs

License
-------

Copyright © 2016-2017 Solus Project, ferryd Developers (See AUTHORS file)

`ferryd` is available under the terms of the Apache-2.0 license
