# httpcat

A `netcat` that understands HTTP

## Installation

```bash
go install gitub.com/pyrsos-dev/httpcat
```

Or

```bash
git clone https://github.com/pyrsos-dev/httpcat.git
cd httpcat
make
su -c 'install -m755 httpcat /usr/bin/httpcat'
```

## Usage

```bash
httpcat

# In another terminal

curl -X 'POST' http://localhost:8080 -d '{"foo": "bar"}'
```

## Options

```
-H string
      alias for -headers
-b string
      alias for -body (default "STDOUT")
-bdelim string
      what to write after writing the request body. (default "\n")
-body string
      where to write the request body. Valid options are STDOUT, STDERR or a path to a file. (default "STDOUT")
-headers string
      where to write the request headers. Valid options are STDOUT, STDERR or a path to a file.
-i string
      alias for -interface (default "127.0.0.1")
-interface string
      network interface to bind to (default "127.0.0.1")
-l string
      alias for -log (default "STDERR")
-log string
      where to write logs. Logs will be discarded if you set any output flag to STDERR. Valid options are STDOUT, STDERR or a path to a file. (default "STDERR")
-p uint
      alias for -port (default 8080)
-port uint
      port to bind to (default 8080)
-verbosity string
      logging verbosity. Valid options are error, warn, info, debug.

```

## Missing features

- Headers support
- TLS support (is this even relevant?)
- Client mode (why would anyone need this? Just use `curl`)
