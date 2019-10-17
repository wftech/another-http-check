# another-http-check

This is replacement for original Nagios `check_http` check plugin. The original plugin contains some bugs and 
provides sometimes misleading error messages.

## Usage


    another-http-check [OPTIONS]


| Application Options:   |                                                                               |
|----------------------|---------------------------------------------------------------------------------|
| `-H=`                | Host ex. google.com                                                             |
| `-I=`                | IPv4 address ex. 8.8.4.4                                                        |
| `-u`, `--uri=`       | URI to check (default: /)                                                       |
| `-p=`                | Port ex. 80 for HTTP 443 for HTTPS (default: 80)                                |
| `-S`, `--tls`        | Use HTTPS                                                                       |
| `-t`, `--timeout=`   | Timeout (default: 30)                                                           |
| `--auth-basic`       | Use HTTP basis                                                                  |
| `--auth-ntlm`        | Use NTLM auth                                                                   |
| `-a`, `--auth=`      | provide  password to authenticate. example `user:password`                      |
| `-e`, `--expect=`    | Expected HTTP code (default: `200)`                                             |
| `-s`, `--string=`    | Search for given string in response body                                        |
| `-C=`                | Check SSL cert expiration                                                       |
| `-k`, `--insecure`   | Controls whether a client verifies the server's certificate chain and host name |
|                      |                                                                                 |
| `-v`, `--verbose`    | Verbose mode                                                                    |
| `--guess-auth`       | Guess auth type (none, basic, NTLM). Generates two requests instead of one      |
| `-h`, `--help`       | Show this help message                                                          |


## Build requirements

- Docker
- make

## How to compile

- `make` creates statically linked binary
- `make test` runs tests
- `make runshell` opens shell inside Docker container (`vim` setup for hacking included)
- `make rpm` - creates RPM package


## Licence

Apache 2
