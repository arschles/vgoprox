---
title: Pre-filling disk storage
description: How to pre-fill the disk cache
weight: 5
---

One of the popular features of Athens is that it can be run completely cut off from the internet. In this case, though, it can't reach out to an upstream (e.g. a VCS or another module proxy) to fetch modules that it doesn't have in storage. So, we need to manually fill up the disk partition that Athens uses with the dependencies that we need.

This document will guide you through packaging up a single module called `github.com/my/module`, and inserting it into the Athens disk storage.

# First, get the tools

You'll need to produce the following assets from module source code:

- `source.zip` - just the Go source code, packaged in a zip file
- `go.mod` - just the `go.mod` file from the module
- `$VERSION.info` - metadata about the module

The `source.zip` file has a specific directory structure and the `$VERSION.info` has a JSON structure, both of which you'll need to get right in order for Athens to serve up the right dependency formats that the Go toolchain will accept.

>We don't recommend that you create these assets yourself. Instead, use [pacmod](https://github.com/plexsystems/pacmod)

To install the `pacmod` tool, run `go get` like this:

```console
$ go get github.com/plexsystems/pacmod
```

This command will install the `pacmod` binary to your `$GOPATH/bin/pacmod` directory, so make sure that is in your `$PATH`.

# Next, run `pacmod` to create assets

After you have `pacmod`, you'll need the module source code that you want to package. Before you run the command, set the `VERSION` variable in your environment to the version of the module you want to generate assets for.

Below is an example for how to configure it.

```console
$ export VERSION="v1.0.0
```

>Note: make sure your `VERSION` variable starts with a `v`

Next, navigate to the top-level directory of the module source code, and run `pacmod` like this:

```console
$ pacmod pack github.com/my/module $VERSION .
```

Once this command is done, you'll notice three new files in the same same directory you ran the command from:

- `go.mod`
- `$VERSION.info`
- `$VERSION.zip`

# Next, move assets into Athens storage directory

Now that you have assets built, you need to move them into the location of the Athens disk storage. In the below commands, we'll assume `$STORAGE_ROOT` is the environment variable that points to the top-level directory that Athens uses for its on-disk.

>If you set up Athens with the `$ATHENS_DISK_STORAGE_ROOT` environment variable, the root of this storage location is the value of this environment variable. Use `export STORAGE_ROOT=$ATHENS_DISK_STORAGE_ROOT` to prepare your environment for the below commands.

First create the subdirectory into which you'll move the assets you created:

```console
$ mkdir -p $STORAGE_ROOT/github.com/my/module/$VERSION
```

Finally, make sure that you're still in the module source repository root directory (the same as you were in when you ran the `pacmod` command), and move your three new files into the new directory you just created:

```console
$ mv go.mod $STORAGE_ROOT/github.com/my/module/$VERSION/go.mod
$ mv $VERSION.info $STORAGE_ROOT/github.com/my/module/$VERSION/$VERSION.info
$ mv $VERSION.zip $STORAGE_ROOT/github.com/my/module/$VERSION/source.zip
```

>Note that we've changed the name of the `.zip` file

# Finally, test your setup

At this point, your Athens server should have its disk-based cache filled with the `github.com/my/module` module at version `$VERSION`. Next time you request this module, Athens will find it in its disk storage and will not try to fetch it from an upstream source.

You can quickly test this behavior by running below `curl` command, assuming your Athens server is running on `http://localhost:3000` and is already configured to use the same disk storage that you pre-filled above.

```console
$ curl localhost:3000/github.com/my/module/@v/$VERSION.info
```

When you run this command, Athens should immediately return, without contacting any other network services.
