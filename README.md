# hydroxide-push
### *Forked from [Hydroxide](https://github.com/emersion/hydroxide)*
[![.github/workflows/build.yaml](https://github.com/0ranki/hydroxide-push/actions/workflows/build.yaml/badge.svg)](https://github.com/0ranki/hydroxide-push/actions/workflows/build.yaml) ![GitHub Release](https://img.shields.io/github/v/release/0ranki/hydroxide-push)

<img src="https://github.com/0ranki/hydroxide-push/assets/50285623/04959566-3d13-4be4-84bd-d7daad3a3166" width="600">

## Push notifications for Proton Mail mobile via a UP provider

Protonmail depends on Google services to deliver push notifications,
This is a stripped down version of [Hydroxide](https://github.com/emersion/hydroxide)
to get notified of new mail. See original repo for details on operation.

Should work with free accounts too.

<sup><sub>Github is used to build the binaries and container images with Github Actions, and host the pre-built releases.
Mirrored from https://git.oranki.net/jarno/hydroxide-push</sub></sup>

Pre-built releases and container images are available on [Github](https://github.com/0ranki/hydroxide-push).


## Setup

Download or build the binary, pull the pre-built container image or build the image yourself.
Simplest way is to run the pre-built container image.

Login and push gateway details are saved under `$HOME/.config/hydroxide`. The container
image saves configuration under `/data`, so mount a named volume or host directory there.
The examples below use a named volume.

If using Docker, substitute `podman` with `docker` in the examples. 

Binary:
```shell
./hydroxide-push auth your.proton@email.address
```
Container:
```shell
podman run -it --rm -v hydroxide-push:/data ghcr.io/0ranki/hydroxide-push auth your.proton@email.address
```
You will be prompted for the Proton account credentials and the details for the push server. Proton credentials are stored encrypted form.

The auth flow generates a separate password for the bridge to fake a login to the bridge, which is stored in plaintext to `$HOME/.config/notify.json`. Unlike upstream `hydroxide`, there is no service listening on any port, the password isn't useful for anything else.

### Reconfigure push server
Binary:
```shell
hydroxide-push setup-ntfy
```
Container:
```shell
podman run -it --rm -v hydroxide-push:/data ghcr.io/0ranki/hydroxide-push setup-ntfy
```
Alternatively to run the command in an already running container (replace `name-of-container` with the name or id of the running container):
```shell
podman exec -it name-of-container /hydroxide-push setup-ntfy
```
You'll be asked for the base URL of the push server, topic then username and password for HTTP basic authentication.
The push endpoint configuration can be changed while the daemon is running.

Username and password are stored in `notify.json`, the password is only Base64-encoded. You should create a dedicated user that
has write-only access to the topic for the daemon.

**Push topic username and password are cleared each time setup-ntfy is run, they need to be entered manually every time.**

The currently configured values are shown inside braces. Leave input blank to use the current values.

### Poll interval

The interval between checking messages can be configured by setting the environment variable `POLL_INTERVAL`.
The value is interpreted as the number of seconds. For example, setting `POLL_INTERVAL=60` will configure the
service to check for new messages every 60 seconds.

The default value is 10 seconds.

### Start the service

Binary:
```shell
hydroxide-push notify
# or
POLL_INTERVAL=30 hydroxide-push notify
```
Container:
```shell
podman run -it --rm -v hydroxide-push:/data ghcr.io/0ranki/hydroxide-push
# or
podman run -it --rm -e POLL_INTERVAL=30 -v hydroxide-push:/data ghcr.io/0ranki/hydroxide-push
```

## Podman pod

A Podman kube YAML file is provided in the repo.

> **Note:** If you're using 2FA or just don't want to put your password to a file, use the manual method above. Make sure the volume name (claimName) in the YAML mathces what you use in the commands. 

- Download/copy `hydroxide-push-podman.yaml` to an empty directory on the machine you intend to run the daemon on
- Edit the config values at the top of the file
- Start the pod:
    ```shell
    podman kube play ./hydroxide-push-podman.yaml
    ```
    - Latest container image is pulled
    - A named volume (`hydroxide-push`) will be created for the configuration
    - Login to Proton and push URL configuration is handled automatically, after which the daemon starts
- After the initial setup, the ConfigMap (before `---`) can be removed from the YAML. Optionally to clear the environment variables, run

    ```shell
    podman kube play ./hydroxide-push-podman.yaml --replace
    ```
    The command can also be used to pull the latest version and restart the pod.
- To reauthenticate or clear data, simply remove the named volume or run the `auth` command

## Updating

Binary:
- stop the service
- download or build the new version, replace the of the old binary
- restart the service

Container:
- pull latest image
- restart container

## Building locally
Clone the repo, then `cd` to the repo root

Binary:
> Requires Go 1.22
```shell
CGO_ENABLED=0 go build -o $HOME/.local/bin/hydroxide-push ./cmd/hydroxide-push/
```
Container:
```shell
podman build -t localhost/hydroxide-push:latest .
```


## License
MIT
