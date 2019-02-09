# dn42regsrv

A REST API for the DN42 registry, written in Go, to provide a bridge between
interactive applications and the DN42 registry.

## Features

- REST API for querying DN42 registry objects
- Able to decorate objects with relationship information based on SCHEMA type definitions
- Includes a simple webserver for delivering static files which can be used to deliver
  basic web applications utilising the API (such as the included DN42 Registry Explorer)
- Automatic pull from the DN42 git repository to keep the registry up to date
- Included responsive web app for exploring the registry

## Building

Requires [git](https://git-scm.com/) and [go](https://golang.org)

```
go get https://git.dn42.us/burble/dn42regsrv
```

## Running

Use --help to view configurable options
```
./dn42regsrv --help
```

The server requires access to a clone of the DN42 registry and for the git executable
to be accessible.  
If you want to use the auto pull feature then the registry must
also be writable by the server.

```
cd ${GOROOT}/src/dn42regsrv
git clone http://git.dn42.us/dn42/registry.git
./dn42regsrv --help
./dn42regsrv
```

A sample service file is included for running the server under systemd

## Using

By default the server will be listening on port 8042.  
See the [API.md](API.md) file for a detailed description of the API.


## Support

Please feel free to raise issues or create pull requests for the project git repository.

## #ToDo

### Server

- Add WHOIS interface
- Add endpoints for ROA data
- Add attribute searches

### DN42 Registry Explorer Web App

- Add search history and fix going back
- Allow for attribute searches

