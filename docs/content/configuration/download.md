---
title: The download mode file
description: What to do when a module is not in storage
weight: 1
---

Athens accepts an [HCL](https://github.com/hashicorp/hcl) formatted file that has instructions for how it should behave when a module@version isn't found in its storage. This functionality gives Athens the flexibility configure Athens to fit your organization's needs. The most popular uses of this download file are:

- Configure Athens to never download or serve a module or group of modules
- Redirect to a different module proxy for a module or group of modules

This document will outline how to use this file - called the download mode file - to accomplish these tasks and more.

>Please see the "Use cases" section below for more details on how to enable these behaviors and more.

## Configuration

First, once you've created your download mode file, you tell Athens to use it by setting the `DownloadMode` configuration parameter in the `config.toml` file, or setting the `ATHENS_DOWNLOAD_MODE` environment variable. You can set this configuration value to one of two values to tell Athens to use your file:

1. Set its value to `file:$FILE_PATH`, where `$FILE_PATH` is the path to the HCL file
2. Set its value to `custom:$BASE_64` where `$BASE_64` is the base64 encoded HCL file

>Instead of one of the above two values, you can set this configuration to `sync`, `async`, `none`, `redirect`, or `async_redirect`. If you do, the download mode will be set globally rather than for specific sub-groups of modules. See below for what each of these values mean.

## Download mode keywords

If Athens receives a request for the module `github.com/pkg/errors` at version `v0.8.1`, and it doesn't have that module and version in its storage, it will consult the download mode file for specific instructions on what action to take:

1. **`sync`**: Synchronously download the module from VCS via `go mod download`, persist it to the Athens storage, and serve it back to the user immediately. Note that this is the default behavior.
2. **`async`**: Return a 404 to the client, and asynchronously download and persist the module@version to storage.
3. **`none`**: Return a 404 and do nothing.
4. **`redirect`**: Redirect to an upstream proxy (such as proxy.golang.org) and do nothing after.
5. **`async_redirect`**: Redirect to an upstream proxy (such as proxy.golang.org) and asynchronously download and persist the module@version to storage.

Athens expects these keywords to be used in conjunction with module patterns (`github.com/pkg/*`, for example). You combine the keyword and the pattern to specify behavior for a specific group of modules.

>Athens uses the Go [path.Match](https://golang.org/pkg/path/#Match) function to parse module patterns.

Below is an example download mode file.

```javascript
downloadURL = "https://proxy.golang.org"

mode = "async_redirect"

download "github.com/gomods/*" {
    mode = "sync"
}

download "golang.org/x/*" {
    mode = "none"
}

download "github.com/pkg/*" {
    mode = "redirect"
    downloadURL = "https://gocenter.io"
}
```

The first two lines describe the _default_ behavior for all modules. This behavior is overridden for select module groups below. In this case, the default behavior is:

- Immediatley redirect all requests to `https://proxy.golang.org`
- In the background, download the module from the version control system (VCS) and store it

The rest of the file contains `download` blocks. These override the default behavior for specific groups of modules.

The first block specifies that any module matching `github.com/gomods/*` (such as `github.com/gomods/athens`) will be downloaded from GitHub, stored, and then returned to the user.

The second block specifies that any module matching `golang.org/x/*` (such as `golang.org/x/text`) will always return a HTTP 404 response code. This behavior ensures that Athens will _never_ store or serve any module names starting with `golang.org/x`.

If a user has their `GOPROXY` environment variable set with a comma separated list, their `go` command line tool will always try the option next in the list. For example, if a user has their `GOPROXY` environment variable set to `https://athens.azurefd.net,direct`, and then runs `go get golang.org/x/text`, they will still download `golang.org/x/text` to their machine. The module just won't come from Athens.

The last block specifies that any module matching `github.com/pkg/*` (such as `github.com/pkg/errors`) will always redirect the `go` tool to https://gocenter.io. In this case, Athens will never persist the given module to its storage.

## Use cases

The download mode file is versatile and allows you to configure Athens in a large variety of different ways. Below are some of the mode common.

## Blocking certain modules

If you're running Athens to serve a team of Go developers, it might be useful to ensure that the team doesn't use a specific group or groups of modules (for example, because of licensing or security issues).

In this case, you would write this in your file:

```hcl
download "bad/module/repo/*" {
    mode = "none"
}
```

### Preventing storage overflow

If you are running Athens using a [storage backend](./storage) that has limited space, you may want to prevent Athens from storing certain groups of modules that take up a lot of space. To avoid exhausting Athens storage, while still ensuring that the users of your Athens server still get access to the modules you can't store, you would use a `redirect` directive, as shown below:

```hcl
download "very/large/*" {
    mode = "redirect"
    url = "https://reliable.proxy.com"
}
```

>If you use the `redirect` mode, make sure that you specify a `url` value that points to a reliable proxy.
