# SwitchTube-Downloader: A Streamlined CLI for SwitchTube Video Downloads

**SwitchTube-Downloader** is a lightweight, efficient command-line tool designed
to easily download videos from [SwitchTube](https://tube.switch.ch/).

## Getting Started

<!-- TODO: Change link to actual release page -->

1. **Download the Binary**: Visit the [releases page](https://github.com/domi413/SwitchTube-Downloader)
   to obtain the appropriate binary for your operating system (Linux, MacOS, or Windows).
   _Note_: You may need to make the binary executable

2. **Explore Usage**: Run `./switch-tube-downloader` to access the help menu,
   which provides clear guidance on available commands.

3. **Obtain an Access Token**: A SwitchTube access token is required. Generate
   one [here](https://tube.switch.ch/access_tokens) to authenticate your
   requests.

```
./switch-tube-downloader
A CLI downloader for SwitchTube videos

Usage:
  SwitchTube-Downloader [command]

Available Commands:
  download    Download a video or channel
  help        Help about any command
  token       Manage the SwitchTube access token
  version     Print the version number of the SwitchTube downloader

Flags:
  -h, --help   help for SwitchTube-Downloader

Use "SwitchTube-Downloader [command] --help" for more information about a command.
```

## Why Choose SwitchTube-Downloader?

While other tools exist for downloading SwitchTube content, **SwitchTube-Downloader** stands out for its compact design, user-friendly interface, and advanced features. Here’s how it compares:

| Feature                        | [SwitchTube-Downloader](https://github.com/domi413/SwitchTube-Downloader) | [switchtube-dl](https://github.com/panmona/switchtube-dl) | [switchtube-rs](https://github.com/jeremystucki/switchtube-rs) |
| ------------------------------ | ------------------------------------------------------------------------- | --------------------------------------------------------- | -------------------------------------------------------------- |
| **Binary Size**                | 6.7MB (Simple and light) ✅                                               | 54.47MB (Bulky)                                           | No binary release                                              |
| **Store Access Token**         | Automatic storage ✅                                                      | Manual configuration                                      | Not supported                                                  |
| **Encrypted Access Token**     | Secure encryption ✅                                                      | No encryption                                             | Not supported                                                  |
| **Intuitive Downloads**        | One simple command ✅                                                     | Separate commands for videos and channels                 | Complex CLI usage                                              |
| **Video download**             | Supported ✅                                                              | Supported ✅                                              | Not supported                                                  |
| **Channel download**           | Supported ✅                                                              | Supported ✅                                              | Supported ✅                                                   |
| **Select videos of a channel** | Supported ✅                                                              | Supported ✅                                              | Not supported                                                  |

Honorable mention: There is yet another SwitchTube downloader also written in go: [switchdl](https://github.com/Erl-koenig/switchdl)

## Testing the SwitchTube API

For developers or curious users, you can interact directly with the SwitchTube API using the following command:

```bash
curl -H "Authorization: Token {your_token}" \
     https://tube.switch.ch/api/v1/xxx
```

E.g., you can write the output to a file to examine the JSON structure:

```bash
curl -H "Authorization: Token cs6UwtHX7DyV2e_CTfS6Bw2twEqeRemiQhNr5Rkt4TU" \
               https://tube.switch.ch/api/v1/browse/channels/{channel_id}/videos | tee tmp.json
```
