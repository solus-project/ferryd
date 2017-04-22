Design considerations
---------------------

Here follows some considerations for the design of binman, to avoid the mistakes of the
current incarnation being used by Solus.

Database
========

The database will be a boltdb key-value store organised into multiple nested buckets for quick and easy access.
In effect, the database will look similar to the following::

    # The default target will be used when a package upload doesn't have an explicitly
    # configured destination. This must always be set
    /default_target: unstable

    /repo:
        /id: unstable
        /package:
            /nano:

                # All available providers under this name
                /available:

                    - nano-2.8.1-69-1-x86_64.eopkg
                    - nano-2.7.5-68-1-x86_64.eopkg
                    - nano-2.7.5-67-1-x86_64.eopkg

                # Delta package map
                /deltas:

                    # Mark the available delta so we know we can use it
                    - nano-67-69-1-x86_64.delta.eopkg: AVAILABLE

                    # Mark failure to delta (too big, etc) so there are no repeat attempts (slow)
                    - nano-68-69-1-x86_64.delta.eopkg: FAILED
                    

                # The version that will be published in the index
                /published: nano-2.8.1-69-1-x86_64.eopkg

                # This allows cheap grouping of packages without having to descend
                # and deserialize, i.e. removing all packages matching a given source.
                /source: nano

    # A stable repo sharing packages with unstable repo
    /repo:
        /id: stable
        /package:
            /nano:
                /available:
                    - nano-2.7.5-67-1-x86_64.eopkg
                /published: nano-2.7.5-67-1-x86_64.eopkg
                /source: nano


    /pool:
        /nano-2.8.1-69-1-x86_64.eopkg:

            # Every reference must contain a schema version to allow migrations
            schema_version: 0.1
            name: nano
            # How many instances of this file are used within the repo
            refcount: 1
            # Raw package data
            metadata: ...
        
        /nano-2.7.5-68-1-x86_64.eopkg:

            schema_version: 1.0
            name: nano
            refcount: 1
            metadata: ...

        /nano-2.7.5-67-1-x86_64.eopkg:

            schema_version: 1.0
            name: nano
            refcount: 2
            metadata: ...

        /nano-67-69-1-x86_64.delta.eopkg:

            schema_version: 1.0
            name: nano
            refcount: 1
            metadata: ...

Thus a lookup for a single package must traverse multiple buckets::

    name := "nano"
    repo := "stable"
    head := /repo:$repo/package:$name/published
    package := /pool:$published

Updating an individual Package
==============================

As a new ``.eopkg`` is introduced, we first pool the asset (``/pool``) and store the
eopkg on disk. This ``.eopkg`` ID is then merged with the available providers for
that given package name (``/available``). We then check that the release number for
the package is higher than the published package for this name, and updating it
if necessary.

At this point we should immediately schedule a write of the repository index (discussed below)
and schedule a delta-map operation. In short, all deltas for a provider that do not point to
the tip version (``/published``) will be scheduled for deletion. We'll then attempt to create
the new deltas for the tip version and mark their status under ``/deltas``. Once the new deltas
have been created (in a parallel routine) we can then reschedule an index to the repository.
In short this ensures that large package updates that land will never freeze/block the repository
waiting for large updates.

Ensuring Availablity
====================

A common issue seen with the older Pythonic implementation of ``binman`` used in Solus, is
reliable availability of the mutating index. This in turn resulted in clients attempting to
read the unstable repository during an index update, which would fail on hash tests and be
determined compromised or corrupt. Additionally, all operations happened in a completely sequential
nature, meaning that delta map, inclusion, and indexing could cause massive delays on the availability
of an update, blocking the build queue and damaging cadence.

In this implementation, channels and worker pools will be utilised to ensure that writing the index
and inclusion of the new eopkg files happens as fast as possible. As soon as they're processed, we
can then walk the repo DB keys and emit a new index. The files will be initially written out with temporary
file names, and then renamed over the existing files. This ensures that we have an atomic update to the index
and new and old clients are never negatively impacted. Once long running operations have completed, such as
delta mapping, we can cheaply write out the new index and expect the packages to become immediately available.
This is in stark contrast of the old method, which did not retain state, thus walked and parsed the underlying
tree of eopkg (in the tens of thousands) to emit the index.

Deduplication
=============

All ``.eopkg`` files are maintained in accordance with a reference count. One thing of
note is that the ``.eopkg`` file names **must** be unique within the whole set of managed
repositories. The files will always be stored within the pool tree, and hardlinked into
the intended repository trees to be available. The reference count in this case will be
incremented, and this is done for both the deltas and and complete packages. In short,
it ensures that files are never recreated unless necessary, thus the disk requirements
are far lower for repository branches, and there is no time wasted on reproducing deltas
on minor syncs.

Upon a deletion of a package from a given repository, the reference count will be decremented,
and the file will be unlinked within the target repository tree. Once the reference count
bottoms out at 0, the file will then be completely removed from the pool tree, and from
the ``/pool`` bucket.

Minimizing Updates
==================

The ``.eopkg`` files arriving from the secure build server should be accompanied by a
transit manifest. There should never be a situation in which a group of packages is
only partially available, i.e. a library package without the accompanying new devel
subpackage, which would introduce broken dependencies.

The manifest will include the expected set of packages, and their hash sums, so that
the repository may confirm a full payload was recieved and has full integrity. Each
upload set is only processed when the full payload has been received. This file shall
be a strongly typed TOML file::

    [manifest]
    version = "1.0"
    target = "unstable" # Optional, use repos default target otherwise.

    [[file]]
    path = "nano-2.7.5-68-1-x86_64.eopkg"
    sha256 = "1810f4d36d42a9d41a37bcd31a70c2279c4cb7b02627bcab981f94f3a24bfcc5"

    [[file]]
    path = "nano-dbginfo-2.7.5-68-1-x86_64.eopkg"
    sha256 = "e25f9326bad558da88e06839249d0a29aaec199995ab85dbd91bfb38913e1b13"

The upload file shall be of the form: ``$source-$version-$release-$arch.tram``, i.e::

    nano-2.7.5-68-x86_64.tram

In turn, the builder will monitor the directory for new changes and attempt to validate
the ``*.tram`` (transit manifest) files on each run. To ensure the maximum efficiency
in processing new uploads, the build server should ensure to send the transit manifest
**after** all ``*.eopkg`` files, which will result in less delays and missing files during
checks, allowing immediate availability of the new package set.
