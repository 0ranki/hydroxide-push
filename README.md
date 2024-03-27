# hydroxide-push
### *Forked from [Hydroxide](https://github.com/emersion/hydroxide)*

## Push notifications for Proton Mail mobile via a UP provider [![.github/workflows/build.yaml](https://github.com/0ranki/hydroxide-push/actions/workflows/build.yaml/badge.svg)](https://github.com/0ranki/hydroxide-push/actions/workflows/build.yaml)

Protonmail depends on Google services to deliver push notifications,
This is a stripped down version of [Hydroxide](https://github.com/emersion/hydroxide)
to get notified of new mail. See original repo for details on operation.

## Setup

Download (soon), build the binary or the container image or pull the container image yourself.
Simplest way is to run the container image.

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
podman run -it --rm -v hydroxide-config:/data ghcr.io/0ranki/hydroxide-push auth your.proton@email.address
```
You will be prompted for the Proton account credentials and the details for the push server. Proton credentials are stored encrypted form.

The auth flow generates a separate password for the bridge, which is stored in plaintext
to `$HOME/.config/notify.json`. Unlike upstream `hydroxide`, there is no service listening on any port,
all communications is internal to the program.

### Reconfigure push server
Binary:
```shell
hydroxide-push setup-ntfy
```
Container:
```shell
podman run -it --rm -v hydroxide-config:/data ghcr.io/0ranki/hydroxide-push setup-ntfy
```
You'll be asked for the base URL of the push server, and the topic. These will probably
be combined to a single string in future versions.

**NOTE:** Authentication for the push endpoint is not yet supported.

### Start the service

Binary:
```shell
hydroxide-push notify
```
Container:
```shell
podman run -it --rm -v hydroxide-config:/data ghcr.io/0ranki/hydroxide-push
```

## License
MIT
