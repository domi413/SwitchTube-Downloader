# SwitchTube-Downloader: A Streamlined CLI for SwitchTube Video Downloads

**SwitchTube-Downloader** is a lightweight, efficient command-line tool designed
to easily download videos from [SwitchTube](https://tube.switch.ch/).

## Getting Started

<!-- TODO: Change link to actual release page -->

1. **Download the binary**: Visit the [releases page](https://github.com/domi413/SwitchTube-Downloader)
   to obtain the appropriate binary for your operating system (Linux, MacOS, Windows).

2. **Make executable**: After downloading, ensure the binary is executable by running:
   ```bash
   chmod +x switch-tube-downloader
   ```
3. **Usage**: Run `./switch-tube-downloader` to access the help menu,
   which provides clear guidance on available commands.

4. **Create access token**: A SwitchTube access token is required. Generate
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

<details>
  <summary>Click for Detailed Usage Instructions</summary>

Running the SwitchTube Downloader without arguments displays available commands:

<pre><code>
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
</code></pre>

## Downloading a video or a channel

To download a video or channel, use the `download` command with either the video/channel ID or its full URL:

<pre><code>./switch-tube-downloader download {id or url}</code></pre>

For example, for the URL `https://tube.switch.ch/channels/dh0sX6Fj1I`, the ID is `dh0sX6Fj1I`. You can use either:

- **URL**: More convenient, directly copied from the browser:
  <pre><code>./switch-tube-downloader download https://tube.switch.ch/channels/dh0sX6Fj1I</code></pre>

- **ID**: Shorter, but requires extracting the ID:
  <pre><code>./switch-tube-downloader download dh0sX6Fj1I</code></pre>

To view detailed help for the `download` command:

<pre><code>
./switch-tube-downloader download --help
Download a video or channel. Automatically detects if input is a video or channel.
You can also pass the whole URL instead of the ID for convenience.

Usage:
SwitchTube-Downloader download <id|url> [flags]

Flags:
-a, --all       Download the whole content of a channel
-e, --episode   Prefixes the video with episode-number e.g. 01_OR_Mapping.mp4
-f, --force     Force overwrite if file already exist
-h, --help      help for download
</code></pre>

### Using Flags

You can add optional flags to customize the download. For example:

- Single flag:
  <pre><code>./switch-tube-downloader download dh0sX6Fj1I -f</code></pre>

- Multiple flags combined:
  <pre><code>./switch-tube-downloader download dh0sX6Fj1I -a -f -e</code></pre>

## Managing Access Tokens

The `token` command manages the SwitchTube access token stored in the system keyring:

<pre><code>
./switch-tube-downloader token
Manage the SwitchTube access token stored in the system keyring

Usage:
  SwitchTube-Downloader token [flags]
  SwitchTube-Downloader token [command]

Available Commands:
  delete      Delete access token from the keyring
  get         Get the current access token
  set         Set a new access token

Flags:
  -h, --help   help for token

Use "SwitchTube-Downloader token [command] --help" for more information about a command.
</code></pre>

**Note**: The `delete` subcommand removes the token without a confirmation prompt, so use it carefully.

</details>

## Why to choose (this) SwitchTube-Downloader?

While other tools exist for downloading SwitchTube content, **SwitchTube-Downloader** stands out for its user-friendly interface, and advanced features. Here’s how it compares:

| Feature                        | [SwitchTube-Downloader](https://github.com/domi413/SwitchTube-Downloader) | [switchtube-dl](https://github.com/panmona/switchtube-dl) | [switchtube-rs](https://github.com/jeremystucki/switchtube-rs) |
| ------------------------------ | ------------------------------------------------------------------------- | --------------------------------------------------------- | -------------------------------------------------------------- |
| **Binary Size**                | 6.7MB (Simple and light) ✅                                               | 54.47MB (Bulky)                                           | No binary release                                              |
| **Store Access Token**         | Automatic storage ✅                                                      | Manual configuration                                      | Not supported                                                  |
| **Encrypted Access Token**     | Secure encryption ✅                                                      | No encryption                                             | Not supported                                                  |
| **Intuitive Downloads**        | One simple command ✅                                                     | Separate commands for videos and channels                 | Complex CLI usage                                              |
| **Video download**             | Supported ✅                                                              | Supported ✅                                              | Not supported                                                  |
| **Channel download**           | Supported ✅                                                              | Supported ✅                                              | Supported ✅                                                   |
| **Select videos of a channel** | Supported ✅                                                              | Supported ✅                                              | Not supported                                                  |
| **Support ID and URL**         | Supported ✅                                                              | Not supported                                             | Not supported                                                  |

Honorable mention: There is yet another SwitchTube downloader also written in go: [switchdl](https://github.com/Erl-koenig/switchdl)

## FAQ

> Why is there no option to define the output directory?

Too many flags can make a command-line tool cumbersome. The current design
focuses on simplicity and ease of use. Also providing a `-o` or `--output` which
could be passed e.g., `switch-tube-downloader download dh0sX6Fj1I -o /path/to/dir`
will result in a complex command structure, which is not user-friendly.

Though, if there is a strong demand for this feature, I might consider adding it.

> Can we select the video quality?

Multiple video quality options are usually available, but to keep the downloader
simple, I chose not to include a quality selection flag, since most users will
use the highest quality available anyway.

## Testing the SwitchTube API

For developers or curious users, you can interact directly with the SwitchTube API using the following command:

```bash
curl -H "Authorization: Token {your_token}" \
        https://tube.switch.ch/api/v1/xxx
```

E.g., you can write the output to a file to examine the JSON structure:

```bash
curl -H "Authorization: Token {your_token}" \
        https://tube.switch.ch/api/v1/browse/channels/{channel_id}/videos | tee tmp.json
```
