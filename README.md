# kubaru

**kubaru (配る: to distribute) is a minimal HTTPS media server for sharing of media files at source quality.
It is designed to work with existing media players as clients, providing a flexible playback experience with minimal configuration.**

---

I often watched movies and shows with friends over voice chat and screen-sharing platforms, but found the experience lacking due to bitrate and resolution constraints.
What if one person wanted to watch with the Japanese audio, while another had a worse opinion? For everyone to have a quality experience, it would often turn into annoying charades of sharing files
or sourcing streams between people of varying technical literacy.

Full-featured media servers seemed overkill for this problem, when I just wanted a lightweight way to share raw media files from my own machine with friends without requiring media indexing, transcoding, or a complicated configuration.
So, I decided to write kubaru.

## Features
- Serve raw media files *as is* through HTTPS, secured with Basic authentication
- Supports automatic generation of a self-signed TLS certificate
- Generates a [`.m3u8` playlist](https://en.wikipedia.org/wiki/M3U) for easy loading into media players
- Allow direct downloads via a browser
- No external dependencies (uses only Go's standard library)

> [!IMPORTANT]
> Due to the simplicity of kubaru and it's limited scope, I feel comfortable exposing it to the internet.
> That being said, I am not responsible for any security issues or data breaches that may occur from running this software!
> **I am continually writing kubaru with security in mind**, and plan to support running it behind a reverse proxy in the near future.

## Installation

Currently, I'm not distributing binaries. Thankfully, it's very straightforward to install Go commands on any platform:

1. [Install Go](https://go.dev/doc/install) and make sure `$GOBIN` is on your path
2. Download and compile kubaru:

	```sh
	go install github.com/pydronia/kubaru@latest
	```

 ## Server Usage

 See `kubaru -h` for all usage options.

 1. Generate a self-signed TLS certificate and key with

	```sh
	kubaru gen-cert -hosts <hosts>
    ```
	Set a list of comma separated IP addresses or domains that you expect your clients to connect from. Don't forget to include `localhost` if you want to connect from your own machine!

	Example: `localhost,<public-ipv4-address>,my-domain.com`

	This will create the files `cert.pem` and `key.pem` in the current directory, which are currently required for the server to run.

	Alternatively, provide your own certificate and key.

2. Run the server

	```sh
	kubaru -port 8888 -path /path/to/media
 	```
	This will start a server listening on all addresses (`[::]`) on port 8888, serving all files in the path provided to `-path`.
	A random password for Basic authentication will be generated and displayed, but these credentials can be set with the `-user` and `-pass` flags,
	or alternatively the `KUBARU_USER` and `KUBARU_PASS` environment variables.

## Client Usage

I recommend using [mpv](https://mpv.io/) for the best support, but connecting to kubaru should work with most modern media players that support HTTPS streams.

Connecting is as simple as opening the following URL in your media player:
```
https://<host>:<port>/playlist.m3u8
```
Most clients will display a warning for the self-signed certificate before prompting for a username and password.

This process can be streamlined by including the username and password in the URL:
```
https://<user>:<pass>@<host>:<port>/playlist.m3u8
```
I recommend the server owner to create a `.m3u8` containing this URL. This file can then be distributed to users and simply opened whenever they want to connect to the server. Note that nesting `.m3u8` playlists in this way may not be supported by all media players.

Alternatively, users can download the playlist file by visiting the link in a browser and then opening the downloaded `.m3u8` file in their player.

If all other options fail to work, media files can be directly accessed and downloaded from the `/files` endpoint using a web browser.

## Future Improvements
- Authentication rate limiting
- Better logging and bandwidth displays
- Custom certificate location
- A simple static home page for instructions for users
- Support for playback syncing! Will require writing player extensions.
