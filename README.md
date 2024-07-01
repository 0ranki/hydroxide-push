# hydroxide-push
### *Forked from [Hydroxide](https://github.com/emersion/hydroxide)*

## Push notifications for Proton Mail mobile via a UP provider [![.github/workflows/build.yaml](https://github.com/0ranki/hydroxide-push/actions/workflows/build.yaml/badge.svg)](https://github.com/0ranki/hydroxide-push/actions/workflows/build.yaml)

![image](https://github.com/0ranki/hydroxide-push/assets/50285623/04959566-3d13-4be4-84bd-d7daad3a3166)

Protonmail depends on Google services to deliver push notifications,
This is a stripped down version of [Hydroxide](https://github.com/emersion/hydroxide)
to get notified of new mail. See original repo for details on operation.

Should work with free accounts too.

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

### Start the service

Binary:
```shell
hydroxide-push notify
```
Container:
```shell
podman run -it --rm -v hydroxide-push:/data ghcr.io/0ranki/hydroxide-push
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


## License
MIT
